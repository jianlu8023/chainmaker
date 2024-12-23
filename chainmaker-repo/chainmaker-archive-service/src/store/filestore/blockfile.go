/*
Copyright (C) BABEC. All rights reserved.


SPDX-License-Identifier: Apache-2.0
*/

// Package filestore define file operation
package filestore

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"chainmaker.org/chainmaker-archive-service/src/archive_utils"
	"chainmaker.org/chainmaker-archive-service/src/interfaces"
	lwsf "chainmaker.org/chainmaker/lws/file"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/tidwall/tinylru"
)

var (
	// ErrCorrupt is returns when the log is corrupt.
	ErrCorrupt = errors.New("log corrupt")

	// ErrClosed is returned when an operation cannot be completed because
	// the log is closed.
	ErrClosed = errors.New("log closed")

	// ErrNotFound is returned when an entry is not found.
	ErrNotFound = errors.New("not found")

	// ErrOutOfOrder is returned from Write() when the index is not equal to
	// LastIndex()+1. It's required that log monotonically grows by one and has
	// no gaps. Thus, the series 10,11,12,13,14 is valid, but 10,11,13,14 is
	// not because there's a gap between 11 and 13. Also, 10,12,11,13 is not
	// valid because 12 and 11 are out of order.
	ErrOutOfOrder = errors.New("out of order")

	// ErrInvalidateIndex 读文件index报错
	ErrInvalidateIndex = errors.New("invalidate rfile index")

	// ErrBlockWrite 写区块文件报错
	ErrBlockWrite = errors.New("write block wfile size invalidate")
)

const (
	dbFileSuffix          = ".fdb"
	lastFileSuffix        = ".END"
	dbFileNameLen         = 20
	blockFilenameTemplate = "%020d"
	gzipMethod            = "gzip"
)

// Options for BlockFile
type Options struct {
	// NoSync disables fsync after writes. This is less durable and puts the
	// log at risk of data loss when there's a server crash.
	NoSync bool
	// SegmentSize of each segment. This is just a target value, actual size
	// may differ. Default is 20 MB.
	SegmentSize int
	// SegmentCacheSize is the maximum number of segments that will be held in
	// memory for caching. Increasing this value may enhance performance for
	// concurrent read operations. Default is 1
	SegmentCacheSize int
	// NoCopy allows for the Read() operation to return the raw underlying data
	// slice. This is an optimization to help minimize allocations. When this
	// option is set, do not modify the returned data because it may affect
	// other Read calls. Default false
	NoCopy bool
	// UseMmap It is a method of memory-mapped rfile I/O. It implements demand
	// paging because rfile contents are not read from disk directly and initially
	// do not use physical RAM at all
	UseMmap bool
	// CompressMethod 支持7z，gzip两种格式
	CompressMethod string // 默认7z，需要安装7z
	// DecompressFileRetainSeconds 默认解压缩文件保留10小时
	DecompressFileRetainSeconds int64 // 默认10 * 3600
	// MaxCompressTimeInSeconds 默认最大压缩解压缩耗时
	MaxCompressTimeInSeconds int // 默认2000
}

// DefaultOptions for Open().
var DefaultOptions = &Options{
	NoSync:                      false,     // Fsync after every write
	SegmentSize:                 67108864,  // 64 MB log segment files.
	SegmentCacheSize:            25,        // Number of cached in-memory segments
	NoCopy:                      false,     // Make a new copy of data for every Read call.
	UseMmap:                     true,      // use mmap for faster write block to file.
	DecompressFileRetainSeconds: 10 * 3600, //默认10×3600 秒回收
	CompressMethod:              "7z",
	MaxCompressTimeInSeconds:    2000,
}

// GetDefaultOptions 获取默认配置
func GetDefaultOptions() *Options {
	return &Options{
		NoSync:                      false,     // Fsync after every write
		SegmentSize:                 67108864,  // 64 MB log segment files.
		SegmentCacheSize:            25,        // Number of cached in-memory segments
		NoCopy:                      false,     // Make a new copy of data for every Read call.
		UseMmap:                     true,      // use mmap for faster write block to file.
		DecompressFileRetainSeconds: 10 * 3600, //默认10×3600 秒回收
		CompressMethod:              "7z",
		MaxCompressTimeInSeconds:    2000,
	}
}

// BlockFile represents a block to rfile
type BlockFile struct {
	mu              sync.RWMutex
	path            string        // absolute path to log directory
	opts            Options       // log options
	closed          bool          // log is closed
	corrupt         bool          // log may be corrupt
	lastSegment     *segment      // last log segment
	lastIndex       uint64        // index of the last entry in log
	sfile           *lockableFile // tail segment rfile handle
	wbatch          Batch         // reusable write batch
	logger          protocol.Logger
	bclock          time.Time
	cachedBuf       []byte
	openedFileCache tinylru.LRU // openedFile entries cache

	compressDir   string                // 压缩文件的存储地址
	decompressDir string                // 解压缩文件的地址
	compressor    interfaces.Compressor // 压缩、解压缩处理类
}

type lockableFile struct {
	sync.RWMutex
	//blkWriter BlockWriter
	wfile lwsf.WalFile
	rfile *os.File
}

// segment represents a single segment rfile.
type segment struct {
	path  string // path of segment rfile
	name  string // name of segment rfile
	index uint64 // first index of segment
	ebuf  []byte // cached entries buffer, storage format of one log entry: checksum|data_size|data
	epos  []bpos // cached entries positions in buffer
}

// bpos denotes index info
// data:        checksum  |  data size  | data
// datasize:    4 byte    |  n,size(n)  | size
// description: pos       |  prefixlen  | end
type bpos struct {
	pos       int // byte position
	end       int // one byte past pos
	prefixLen int
}

// FileStartEndHeight 区块文件信息
type FileStartEndHeight struct {
	BeginHeight  uint64 // 文件的开始区块
	EndHeight    uint64 // 文件的结束区块
	NeedRecordDB bool   // 一个文件完整的，需要记录到数据库
	FileInfo     string // 暂时用不到，后面可能会存放文件信息
}

// Open a new write ahead log
// @param path
// @param compressDir
// @param deCompressDir
// @param opts
// @param logger
// @param BlockFile
// @return error
func Open(path, compressDir, deCompressDir string, opts *Options, logger protocol.Logger) (*BlockFile, error) {
	if opts == nil {
		opts = DefaultOptions
	}
	if opts.SegmentCacheSize <= 0 {
		opts.SegmentCacheSize = DefaultOptions.SegmentCacheSize
	}
	if opts.SegmentSize <= 0 {
		opts.SegmentSize = DefaultOptions.SegmentSize
	}
	if opts.MaxCompressTimeInSeconds <= 0 {
		opts.MaxCompressTimeInSeconds = 2000
	}
	var err error
	if path, err = filepath.Abs(path); err != nil {
		return nil, err
	}
	if compressDir, err = filepath.Abs(compressDir); err != nil {
		return nil, err
	}
	if deCompressDir, err = filepath.Abs(deCompressDir); err != nil {
		return nil, err
	}

	l := &BlockFile{
		path:          path,
		opts:          *opts,
		logger:        logger,
		cachedBuf:     make([]byte, 0, int(float32(opts.SegmentSize)*float32(1.5))),
		decompressDir: deCompressDir,
		compressDir:   compressDir,
	}
	// 初始化压缩/解压缩功能
	if opts.CompressMethod == gzipMethod {
		l.compressor = &CompressGzip{suffix: gzipMethod}
	} else {
		l.compressor = &Compress7z{suffix: "7z", maxTime: opts.MaxCompressTimeInSeconds}
	}

	l.openedFileCache.Resize(l.opts.SegmentCacheSize)
	if err = os.MkdirAll(path, 0777); err != nil {
		return nil, err
	}
	if err = os.MkdirAll(deCompressDir, 0777); err != nil {
		return nil, err
	}
	if err = os.MkdirAll(compressDir, 0777); err != nil {
		return nil, err
	}
	if err = l.load(); err != nil {
		return nil, err
	}
	return l, nil
}

// 将segfile 缓存到cache中,如果有缓存被清除，将被置换清除的读文件关闭
// @receiver l
// @param path
// @param segFile
// @return
func (l *BlockFile) pushCache(path string, segFile *lockableFile) {
	l.logger.Debugf("pushCache begin path %s", path)
	if strings.HasSuffix(path, lastFileSuffix) {
		// has lastFileSuffix mean this is an writing rfile, only happened when add new block entry to rfile,
		// we do not use ofile in other place, since we read new added block entry from memory (l.lastSegment.ebuf),
		// so we should not cache this rfile object
		return
	}
	_, _, _, v, evicted :=
		l.openedFileCache.SetEvicted(path, segFile)
	if evicted {
		// nolint
		if v == nil {
			return
		}
		if lfile, ok := v.(*lockableFile); ok {
			lfile.Lock()
			_ = lfile.rfile.Close()
			l.logger.Debugf("pushCache close path %s closed %s ", path, lfile.rfile.Name())
			lfile.Unlock()
		}
	}
}

// nolint load all the segments. This operation also cleans up any START/END segments.
// @receiver l
// @return error
func (l *BlockFile) load() error {
	var err error
	if err = l.loadFromPath(l.path); err != nil {
		return err
	}
	// for the first time to start
	if l.lastSegment == nil {
		// Create a new log
		segName := l.segmentName(1)
		l.lastSegment = &segment{
			name:  segName,
			index: 1,
			path:  l.segmentPathWithENDSuffix(segName),
			ebuf:  l.cachedBuf[:0],
		}
		l.lastIndex = 0
		l.sfile, err = l.openWriteFile(l.lastSegment.path)
		if err != nil {
			return err
		}
		return err
	}
	l.logger.Debugf("load lastsegment index %d , epos length %d , lastIndex %d",
		l.lastSegment.index, len(l.lastSegment.epos), l.lastIndex)
	// Open the last segment for appending
	if l.sfile, err = l.openWriteFile(l.lastSegment.path); err != nil {
		return err
	}
	// Customize part start
	// Load the last segment, only load uncorrupted log entries
	if err = l.loadSegmentEntriesForRestarting(l.lastSegment); err != nil {
		return err
	}

	// Customize part end
	l.lastIndex = l.lastSegment.index + uint64(len(l.lastSegment.epos)) - 1
	l.logger.Debugf("load after restarting lastsegment index %d , epos length %d , lastIndex %d",
		l.lastSegment.index, len(l.lastSegment.epos), l.lastIndex)
	return nil
}

// loadFromPath 从指定目录加载segment/文件
// @receiver l
// @param path
// @return error
func (l *BlockFile) loadFromPath(path string) error {
	fis, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	//endIdx := -1
	var index uint64
	// during the restart, wal files are loaded to log.segments
	for _, fi := range fis {
		name := fi.Name()
		if fi.IsDir() || len(name) != 24+len(dbFileSuffix) {
			continue
		}
		index, err = strconv.ParseUint(name[0:dbFileNameLen], 10, 64) // index most time is the first height of bfdb rfile
		if err != nil || index == 0 {
			continue
		}
		if strings.HasSuffix(name, lastFileSuffix) {
			segName := name[:dbFileNameLen]
			l.lastSegment = &segment{
				name:  segName,
				index: index,
				path:  l.segmentPathWithENDSuffix(segName),
				ebuf:  l.cachedBuf[:0],
			}
			break
		}
	}
	return nil
}

// openReadFile 从执行目录打开文件，放到lru缓存，将打开文件游标放到文件末尾
//
//	evited文件关闭？
//
// @receiver l
// @param path
// @return lockableFile
// @return error
func (l *BlockFile) openReadFile(path string) (*lockableFile, error) {
	// Open the appropriate rfile as read-only.
	var (
		err     error
		isExist bool
		rfile   *os.File
		ofile   *lockableFile
	)

	fileV, isOK := l.openedFileCache.Get(path)
	if isOK && fileV != nil {
		if isExist, _ = PathExists(path); isExist {
			ofil, ok := fileV.(*lockableFile)
			if ok {
				l.pushCache(path, ofil)
				return ofil, nil
			}
		}
	}

	if isExist, err = PathExists(path); err != nil {
		return nil, err
	} else if !isExist {
		return nil, fmt.Errorf("bfdb rfile:%s missed", path)
	}

	rfile, err = os.OpenFile(path, os.O_RDONLY, 0666)
	if err != nil {
		return nil, err
	}
	l.logger.Debugf("openReadFile path %s", path)
	if _, err = rfile.Seek(0, 2); err != nil {
		return nil, err
	}

	ofile = &lockableFile{rfile: rfile}
	l.pushCache(path, ofile)
	return ofile, nil
}

// openWriteFile 打开写文件
// @receiver l
// @param path
// @return lockableFile
// @return error
func (l *BlockFile) openWriteFile(path string) (*lockableFile, error) {
	// Open the appropriate rfile as read-only.
	var (
		err     error
		isExist bool
		wfile   lwsf.WalFile
		ofile   *lockableFile
	)

	if isExist, err = PathExists(path); err != nil {
		return nil, err
	}

	if !isExist && !strings.HasSuffix(path, lastFileSuffix) {
		return nil, fmt.Errorf("bfdb wfile:%s missed", path)
	}

	if l.opts.UseMmap {
		if wfile, err = lwsf.NewMmapFile(path, l.opts.SegmentSize); err != nil {
			return nil, err
		}
	} else {
		if wfile, err = lwsf.NewFile(path); err != nil {
			return nil, err
		}
	}
	ofile = &lockableFile{wfile: wfile}
	return ofile, nil
}

// segmentName returns a 20-byte textual representation of an index
// for lexical ordering. This is used for the rfile names of log segments.
// @receiver l
// @param index
// @return string
func (l *BlockFile) segmentName(index uint64) string {
	return fmt.Sprintf(blockFilenameTemplate, index)
}

// segmentPath 文件名
// @receiver l
// @param isDeCompressed
// @param name
// @return string
func (l *BlockFile) segmentPath(isDeCompressed bool, name string) string {
	if isDeCompressed {
		return fmt.Sprintf("%s%s", filepath.Join(l.decompressDir, name), dbFileSuffix)
	}
	return fmt.Sprintf("%s%s", filepath.Join(l.path, name), dbFileSuffix)
}

// segmentPathWithENDSuffix 带后缀文件名
// @receiver l
// @param name
// @return string
func (l *BlockFile) segmentPathWithENDSuffix(name string) string {
	if strings.HasSuffix(name, lastFileSuffix) {
		return name
	}
	// 正在写的文件肯定不在解压缩文件夹下面
	return fmt.Sprintf("%s%s", l.segmentPath(false, name), lastFileSuffix)
}

// Close the log.
// sync && close write file, and resize BlockFile's cache size
// @receiver l
// @return error
func (l *BlockFile) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		if l.corrupt {
			return ErrCorrupt
		}
		return ErrClosed
	}
	if err := l.sfile.wfile.Sync(); err != nil {
		return err
	}
	if err := l.sfile.wfile.Close(); err != nil {
		return err
	}
	l.closed = true
	if l.corrupt {
		return ErrCorrupt
	}

	l.openedFileCache.Resize(l.opts.SegmentCacheSize)
	return nil
}

// Write an entry to the block rfile db.
// return block's file name , offset in file , block data size
// lock BlockFile ,sequence write
// @receiver l
// @param index
// @param data
// @return string
// @return uint64
// @return uint64
// @return uint64
// @return uint64
// @return bool
// @return error
func (l *BlockFile) Write(index uint64, data []byte) (fileName string,
	offset, blkLen uint64, startHeight, endHeight uint64, needRecordDB bool, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.bclock = time.Now()
	if l.corrupt {
		return "", 0, 0, 0, 0, false, ErrCorrupt
	} else if l.closed {
		return "", 0, 0, 0, 0, false, ErrClosed
	}
	l.wbatch.Write(index, data)
	bwi, detail, err := l.writeBatch(&l.wbatch)
	if err != nil {
		return "", 0, 0, 0, 0, false, err
	}
	return bwi.FileName, bwi.Offset, bwi.ByteLen, detail.BeginHeight, detail.EndHeight, detail.NeedRecordDB, nil
}

// Cycle the old segment for a new segment.
// 1. flush data to file , close the old and rename the old
// 2. open new file and set BlockFile->lastSegment
// @receiver l
// @return error
func (l *BlockFile) cycle() error {
	if l.opts.NoSync {
		if err := l.sfile.wfile.Sync(); err != nil {
			return err
		}
	}
	if err := l.sfile.wfile.Close(); err != nil {
		return err
	}

	// remove pre last segment name's  end suffix
	orgPath := l.lastSegment.path
	if !strings.HasSuffix(orgPath, lastFileSuffix) {
		return fmt.Errorf("last segment rfile dot not end with %s", lastFileSuffix)
	}
	finalPath := orgPath[:len(orgPath)-len(lastFileSuffix)]
	if err := os.Rename(orgPath, finalPath); err != nil {
		return err
	}
	// cache the previous lockfile
	//l.pushCache(finalPath, l.sfile)

	//  power down issue
	segName := l.segmentName(l.lastIndex + 1)
	s := &segment{
		name:  segName,
		index: l.lastIndex + 1,
		path:  l.segmentPathWithENDSuffix(segName),
		ebuf:  l.cachedBuf[:0],
	}
	var err error
	if l.sfile, err = l.openWriteFile(s.path); err != nil {
		return err
	}
	l.lastSegment = s
	return nil
}

// append 拼接一个entry
// append data to dst , return dst and pos info
// @receiver l
// @param dst
// @param data
// @return []byte
// @return bpos
func (l *BlockFile) appendBinaryEntry(dst []byte, data []byte) (out []byte, epos bpos) {
	// checksum + data_size + data
	pos := len(dst)
	// Customize part start
	dst = appendChecksum(dst, archive_utils.NewCRC(data).Value())
	// Customize part end
	dst = appendUvarint(dst, uint64(len(data)))
	prefixLen := len(dst) - pos
	dst = append(dst, data...)
	return dst, bpos{pos, len(dst), prefixLen}
}

// appendChecksum  Customize part start
// @param dst
// @param checksum
// @return []byte
func appendChecksum(dst []byte, checksum uint32) []byte {
	dst = append(dst, []byte("0000")...)
	binary.LittleEndian.PutUint32(dst[len(dst)-4:], checksum)
	return dst
}

// appendUvarint Customize part end
// x 整型序列化
// @param dst
// @param x
// @return []byte
func appendUvarint(dst []byte, x uint64) []byte {
	var buf [10]byte
	n := binary.PutUvarint(buf[:], x)
	dst = append(dst, buf[:n]...)
	return dst
}

// Batch of entries. Used to write multiple entries at once using WriteBatch().
type Batch struct {
	entry batchEntry
	data  []byte
}

type batchEntry struct {
	index uint64
	size  int
}

// Write an entry to the batch
// @receiver b
// @param index
// @param data
// @return
func (b *Batch) Write(index uint64, data []byte) {
	b.entry = batchEntry{index, len(data)}
	b.data = data
}

// writeBatch 写一个batch 返回索引信息
// 将数据写入到BlockFile中lastSegment所指的文件当中，顺序写入，写文件的时候加了一把锁
// @receiver l
// @param b
// @return StoreInfo
// @return FileStartEndHeight
// @return error
func (l *BlockFile) writeBatch(b *Batch) (*storePb.StoreInfo, *FileStartEndHeight, error) {
	// check that indexes in batch are same
	if b.entry.index != l.lastIndex+uint64(1) {
		l.logger.Errorf(fmt.Sprintf("out of order, b.entry.index: %d and l.lastIndex+uint64(1): %d",
			b.entry.index, l.lastIndex+uint64(1)))
		if l.lastIndex == 0 {
			l.logger.Errorf("your block rfile db is damaged or not use this feature before, " +
				"please check your disable_block_file_db setting in chainmaker.yml")
		}
		return nil, nil, ErrOutOfOrder
	}
	// load the tail segment
	s := l.lastSegment
	var detailInfo FileStartEndHeight
	if len(s.ebuf) > l.opts.SegmentSize {
		// 计算第一个和最后一个区块的高度
		beginHeight := s.index - 1
		endHeight := beginHeight + uint64(len(s.epos)) - 1
		detailInfo.BeginHeight = beginHeight
		detailInfo.EndHeight = endHeight
		detailInfo.NeedRecordDB = true
		// tail segment has reached capacity. Close it and create a new one.
		if err := l.cycle(); err != nil {
			return nil, &detailInfo, err
		}
		// 这里将起始，结束位置和文件的映射存放起来
		s = l.lastSegment
		l.logger.Debugf("writeBatch.file batch-entry-index %d, detail-begin %d , detail-end %d ,"+
			"lastsegment.index %d, lastsegment.epos.len %d , lastsegment.name %s ,lastIndex %d ",
			b.entry.index, detailInfo.BeginHeight, detailInfo.EndHeight, s.index, len(s.epos), s.name, l.lastIndex)
	}

	var epos bpos
	s.ebuf, epos = l.appendBinaryEntry(s.ebuf, b.data)
	s.epos = append(s.epos, epos)

	// startTime := time.Now()
	l.sfile.Lock()
	if _, err := l.sfile.wfile.WriteAt(s.ebuf[epos.pos:epos.end], int64(epos.pos)); err != nil {
		l.logger.Errorf("write rfile: %s in %d err: %v", s.path, s.index+uint64(len(s.epos)), err)
		return nil, nil, err
	}
	l.lastIndex = b.entry.index
	l.sfile.Unlock()
	// l.logger.Debugf("writeBatch block[%d] rfile.WriteAt time: %v", l.lastIndex, ElapsedMillisSeconds(startTime))
	l.logger.Debugf("writeBatch.batch  batch-entry-index %d, detail-begin %d , detail-end %d ,"+
		"lastsegment.index %d, lastsegment.epos.len %d , lastsegment.name %s ,lastIndex %d ",
		b.entry.index, detailInfo.BeginHeight, detailInfo.EndHeight, s.index, len(s.epos), s.name, l.lastIndex)
	if !l.opts.NoSync {
		if err := l.sfile.wfile.Sync(); err != nil {
			return nil, nil, err
		}
	}
	if epos.end-epos.pos != b.entry.size+epos.prefixLen {
		return nil, nil, ErrBlockWrite
	}
	return &storePb.StoreInfo{
		FileName: l.lastSegment.name[:dbFileNameLen],
		Offset:   uint64(epos.pos + epos.prefixLen),
		ByteLen:  uint64(b.entry.size),
	}, &detailInfo, nil
}

// LastIndex returns the index of the last entry in the log. Returns zero when
// log has no entries. when read data ,lock
// @receiver l
// @return uint64
// @return error
func (l *BlockFile) LastIndex() (index uint64, err error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.corrupt {
		return 0, ErrCorrupt
	} else if l.closed {
		return 0, ErrClosed
	}
	if l.lastIndex == 0 {
		return 0, nil
	}
	return l.lastIndex, nil
}

// loadSegmentEntriesForRestarting loads ebuf and epos in the segment when restarting
// @receiver l
// @param s
// @return error
func (l *BlockFile) loadSegmentEntriesForRestarting(s *segment) error {
	data, err := ioutil.ReadFile(s.path)
	if err != nil {
		return err
	}
	var (
		epos []bpos
		pos  int
	)
	ebuf := data
	for exidx := s.index; len(data) > 0; exidx++ {
		var n, prefixLen int
		n, prefixLen, err = loadNextBinaryEntry(data)
		// if there are corrupted log entries, the corrupted and subsequent data are discarded
		if err != nil {
			break
		}
		data = data[n:]
		epos = append(epos, bpos{pos, pos + n, prefixLen})
		pos += n
	}
	// load uncorrupted data, assign one time
	s.ebuf = ebuf[0:pos]
	s.epos = epos
	return nil
}

// loadNextBinaryEntry 读取entry,返回
// @param data
// @return int
// @return int
// @return error
func loadNextBinaryEntry(data []byte) (n, prefixLen int, err error) {
	// Customize part start
	// checksum + data_size + data
	// checksum read
	checksum := binary.LittleEndian.Uint32(data[:4])
	// binary read
	data = data[4:]
	// Customize part end
	size, n := binary.Uvarint(data)
	if n <= 0 {
		return 0, 0, ErrCorrupt
	}
	if uint64(len(data)-n) < size {
		return 0, 0, ErrCorrupt
	}
	// Customize part start
	// verify checksum
	if checksum != archive_utils.NewCRC(data[n:uint64(n)+size]).Value() {
		return 0, 0, ErrCorrupt
	}
	prefixLen = 4 + n
	return prefixLen + int(size), prefixLen, nil
	// Customize part end
}

// ReadLastSegSection an entry from the log. Returns a byte slice containing the data entry.
// return block data , block data 's file name ,block data's begin offset, block data's length
// @receiver l
// @param index
// @return []byte
// @return string
// @return uint64
// @return uint64
// @return error
func (l *BlockFile) ReadLastSegSection(index uint64) (data []byte,
	fileName string, offset uint64, byteLen uint64, err error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.corrupt {
		return nil, "", 0, 0, ErrCorrupt
	} else if l.closed {
		return nil, "", 0, 0, ErrClosed
	}

	s := l.lastSegment
	if index == 0 || index < s.index || index > l.lastIndex {
		return nil, "", 0, 0, ErrNotFound
	}
	epos := s.epos[index-s.index]
	edata := s.ebuf[epos.pos:epos.end]

	// Customize part start
	// checksum read
	checksum := binary.LittleEndian.Uint32(edata[:4])
	// binary read
	edata = edata[4:]
	// Customize part end
	size, n := binary.Uvarint(edata)
	if n <= 0 {
		return nil, "", 0, 0, ErrCorrupt
	}
	if uint64(len(edata)-n) < size {
		return nil, "", 0, 0, ErrCorrupt
	}
	// Customize part start
	if checksum != archive_utils.NewCRC(edata[n:]).Value() {
		return nil, "", 0, 0, ErrCorrupt
	}
	// Customize part end
	if l.opts.NoCopy {
		data = edata[n : uint64(n)+size]
	} else {
		data = make([]byte, size)
		copy(data, edata[n:])
	}
	fileName = l.lastSegment.name[:dbFileNameLen]
	offset = uint64(epos.pos + epos.prefixLen)
	byteLen = uint64(len(data))
	return data, fileName, offset, byteLen, nil
}

// ReadFileSection an entry from the log. Returns a byte slice containing the data entry.
// @receive l
// @param isDeCompressed
// @param fiIndex
// @return []byte
// @return error
func (l *BlockFile) ReadFileSection(isDeCompressed bool, fiIndex *storePb.StoreInfo) ([]byte, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if fiIndex == nil || len(fiIndex.FileName) != dbFileNameLen || fiIndex.ByteLen == 0 {
		l.logger.Warnf("invalidate rfile index: %s", FileIndexToString(fiIndex))
		return nil, ErrInvalidateIndex
	}

	path := l.segmentPath(isDeCompressed, fiIndex.FileName)
	if fiIndex.FileName == l.lastSegment.name[:dbFileNameLen] {
		path = l.segmentPathWithENDSuffix(fiIndex.FileName)
		tmpData := l.lastSegment.ebuf
		if len(tmpData) >= int(fiIndex.Offset)+int(fiIndex.ByteLen) {
			data := tmpData[fiIndex.Offset : fiIndex.Offset+fiIndex.ByteLen]
			return data, nil
		}
	}

	lfile, err := l.openReadFile(path)
	if err != nil {
		return nil, err
	}

	data := make([]byte, fiIndex.ByteLen)
	// 顺序读，加锁
	lfile.RLock()
	n, err1 := lfile.rfile.ReadAt(data, int64(fiIndex.Offset))
	lfile.RUnlock()
	if err1 != nil {
		return nil, err1
	}
	if uint64(n) != fiIndex.ByteLen {
		errMsg := fmt.Sprintf("read block rfile size invalidate, wanted: %d, actual: %d", fiIndex.ByteLen, n)
		return nil, errors.New(errMsg)
	}
	return data, nil
}

// ClearCache clears the segment cache
// lock BlockFile
// @receive l
// @return error
func (l *BlockFile) ClearCache() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.corrupt {
		return ErrCorrupt
	} else if l.closed {
		return ErrClosed
	}
	l.clearCache()
	return nil
}

// clearCache 清除cache
// 关闭BlockFile->openedFileCache 所有的读文件
func (l *BlockFile) clearCache() {
	l.openedFileCache.Range(func(_, v interface{}) bool {
		// nolint
		if v == nil {
			return true
		}
		if s, ok := v.(*lockableFile); ok {
			s.Lock()
			_ = s.rfile.Close()
			s.Unlock()
		}
		return true
	})
	l.openedFileCache = tinylru.LRU{}
	l.openedFileCache.Resize(l.opts.SegmentCacheSize)
}

// Sync performs an fsync on the log. This is not necessary when the
// NoSync option is set to false.
// @receive l
// @return error
func (l *BlockFile) Sync() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.corrupt {
		return ErrCorrupt
	} else if l.closed {
		return ErrClosed
	}
	return l.sfile.wfile.Sync()
}

// TruncateFront 清除数据
func (l *BlockFile) TruncateFront(index uint64) error {
	return nil
}

// FileIndexToString 索引转化为可读字符串
// @param fiIndex
// @return string
func FileIndexToString(fiIndex *storePb.StoreInfo) string {
	if fiIndex == nil {
		return "rfile index is nil"
	}

	return fmt.Sprintf("fileIndex: fileName: %s, offset: %d, byteLen: %d",
		fiIndex.GetFileName(), fiIndex.GetOffset(), fiIndex.GetByteLen())
}

// CheckDecompressFileExist 检查fileName文件是否在解压缩文件夹中存在，返回上次访问文件的unix时间戳
// 仅适用于linux平台
// filename为00000 00000 00000 00001
// @receiver l
// @param fileName
// @return bool
// @return int64
// @return error
func (l *BlockFile) CheckDecompressFileExist(fileName string) (bool, int64, error) {
	filePath := l.segmentPath(true, fileName)
	return l.checkFileExist(filePath)
}

// checkFileExist 检查文件是否存在,上次访问时间
// @receiver l
// @param filePath
// @return bool
// @return int64
// @return error
func (l *BlockFile) checkFileExist(filePath string) (bool, int64, error) {
	file, fileError := os.Stat(filePath)
	if fileError != nil {
		if errors.Is(fileError, os.ErrNotExist) {
			return false, 0, nil
		}
		return false, 0, fileError
	}
	sys, _ := file.Sys().(*syscall.Stat_t)
	return true, sys.Atim.Sec, nil
}

// DeCompressFile 解压compressFileName
// @receiver l
// @param compressFileName
// @return string
// @return error
func (l *BlockFile) DeCompressFile(compressFileName string) (string, error) {
	//compressFile :=
	sourceName := l.constructCompressFileName(false, compressFileName)
	return l.compressor.DeCompressFile(l.compressDir, sourceName, l.decompressDir)
}

// constructCompressFileName 根据传过来的
// @receiver l
// @param needPath
// @param fileName
// @return string
func (l *BlockFile) constructCompressFileName(needPath bool, fileName string) string {
	compressSuffix := "7z"
	if l.opts.CompressMethod == gzipMethod {
		compressSuffix = gzipMethod
	}
	if !needPath {
		// 返回 00000 00000 00000 00001.fdb.7z(gzip)
		return fmt.Sprintf("%s%s.%s", fileName, dbFileSuffix, compressSuffix)
	}
	return fmt.Sprintf("%s%s.%s", filepath.Join(l.compressDir, fileName), dbFileSuffix, compressSuffix)
}

// GetCanCompressHeight 这个函数分析lastsegment的index，
// 这个函数调用必须在加载最新的fdb文件之后才可以执行
// 如果是服务新起来，返回0
// @receiver l
// @return uint64
func (l *BlockFile) GetCanCompressHeight() uint64 {
	if l.lastSegment == nil || l.lastSegment.index <= 1 {
		return 0
	}
	return l.lastSegment.index - 2 //
}

// CompressFileByStartHeight 压缩指定高度起始的文件
// @receiver l
// @param startHeight
// @return string
// @return error
func (l *BlockFile) CompressFileByStartHeight(startHeight uint64) (string, error) {
	fileName := fmt.Sprintf("%s%s", l.segmentName(startHeight+1), dbFileSuffix)
	retSegName := l.segmentName(startHeight + 1)
	return retSegName, l.compressor.CompressFile(l.path, fileName, l.compressDir)
}

// TryRemoveFile 删除文件
// @receiver l
// @param fileName
// @param isDeCompressed
// @return bool
// @return error
func (l *BlockFile) TryRemoveFile(fileName string, isDeCompressed bool) (bool, error) {
	nowUnix := time.Now().Unix()
	path := l.segmentPath(isDeCompressed, fileName)
	exists, lastAccessTs, existsErr := l.checkFileExist(path)
	if existsErr != nil {
		l.logger.Errorf("tryRemoveFile %s ,isDecompressed %T path exist error %s",
			fileName, isDeCompressed, existsErr.Error())
		return false, existsErr
	}
	l.logger.Infof("TryRemoveFile fileName %s , path %s , exists %+v, lastAccess %d , nowUnix %d",
		fileName, path, exists, lastAccessTs, nowUnix)
	if !exists { // 如果不存在，那么直接返回即可
		return false, nil
	}
	// 检查一下文件上次的访问时间，
	if (nowUnix - lastAccessTs) <= l.opts.DecompressFileRetainSeconds {
		//上次访问时间如果小于保留时间，直接返回即可
		return false, nil
	}
	// 检查缓存，关闭文件，删除文件
	ofile, ofileExist := l.openedFileCache.Get(path)
	if ofileExist {
		if lfile, ok := ofile.(*lockableFile); ok {
			lfile.Lock()
			_ = lfile.rfile.Close()
			l.openedFileCache.Delete(path) // 删除缓存
			lfile.Unlock()
		}
	}
	// 下面可以把这个文件删除了
	deleteError := os.Remove(path)
	if deleteError != nil {
		l.logger.Errorf("remove file %s ,isDecompressed %T got error %s ", path, isDeCompressed, deleteError.Error())
		return false, deleteError
	}
	return true, nil
}
