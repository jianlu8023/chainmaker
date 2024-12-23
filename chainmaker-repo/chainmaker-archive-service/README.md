# 编译服务
export GODEBUG=x509ignoreCN=0 && go build -o 
chainmaker-archive-service


# 文件说明 ：
## 生成根证书  
```powershell
# 使用openssl生成根证书的key 
openssl ecparam -genkey -name secp384r1 -out baec-root.key 
# 使用私钥生成根证书 
openssl req -new -x509 -sha256 -key baec-root.key -out baec-root.pem
####################################################
You are about to be asked to enter information that will be incorporated
into your certificate request.
What you are about to enter is what is called a Distinguished Name or a DN.
There are quite a few fields but you can leave some blank
For some fields there will be a default value,
If you enter '.', the field will be left blank.
-----
Country Name (2 letter code) [AU]:
State or Province Name (full name) [Some-State]:
Locality Name (eg, city) []:
Organization Name (eg, company) [Internet Widgits Pty Ltd]:
Organizational Unit Name (eg, section) []:
Common Name (e.g. server FQDN or YOUR name) []: baec-root
Email Address []:
 
```
## 生成服务端的密钥和证书
```powershell
# 生成服务私钥
openssl ecparam -genkey -name secp384r1 -out archive-server.key

# 使用服务私钥生成签发证书请求  
openssl req -new -key archive-server.key -out archive-server.csr

##############################################
You are about to be asked to enter information that will be incorporated
into your certificate request.
What you are about to enter is what is called a Distinguished Name or a DN.
There are quite a few fields but you can leave some blank
For some fields there will be a default value,
If you enter '.', the field will be left blank.
-----
Country Name (2 letter code) [AU]:
State or Province Name (full name) [Some-State]:
Locality Name (eg, city) []:
Organization Name (eg, company) [Internet Widgits Pty Ltd]:
Organizational Unit Name (eg, section) []:
Common Name (e.g. server FQDN or YOUR name) []:baec-archive
Email Address []:

Please enter the following 'extra' attributes
to be sent with your certificate request
A challenge password []:
An optional company name []:

##############################################
# 使用根证书和根密钥对证书请求进行签发，输出一年有效的证书
openssl x509 -req -sha256 -CA baec-root.pem -CAkey baec-root.key -CAcreateserial -days 3650 -in archive-server.csr -out archive-server.pem
```

## 生成客户端的密钥和证书  
```powershell

# 生成客户端私钥  
openssl ecparam -genkey -name secp384r1 -out archive-client.key  

# 使用客户端私钥生成签发证书请求  
openssl req -new -key archive-client.key -out archive-client.csr

#####################################################
You are about to be asked to enter information that will be incorporated
into your certificate request.
What you are about to enter is what is called a Distinguished Name or a DN.
There are quite a few fields but you can leave some blank
For some fields there will be a default value,
If you enter '.', the field will be left blank.
-----
Country Name (2 letter code) [AU]:
State or Province Name (full name) [Some-State]:
Locality Name (eg, city) []:
Organization Name (eg, company) [Internet Widgits Pty Ltd]:
Organizational Unit Name (eg, section) []:
Common Name (e.g. server FQDN or YOUR name) []:baec-archive-client
Email Address []:

Please enter the following 'extra' attributes
to be sent with your certificate request
A challenge password []:
An optional company name []:

# 使用根密钥和根证书对客户端申请签发证书  
openssl x509 -req -sha256 -CA baec-root.pem -CAkey baec-root.key -CAcreateserial -days 3650 -in archive-client.csr -out archive-client.pem
```

