"""
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

"""
import base64
import json
import string
import sys
import unittest
import re

sys.path.append("..")

import config.public_import as gl
from utils.cmc_tools_contract import ContractDeal
from utils.cmc_tools_query import get_user_addr
from utils.cmc_command import Command


class Test(unittest.TestCase):
    def test_balance_a_compare_cert(self):
        print("query UserA address: org1 admin".center(50, "="))
        user_a_address = get_user_addr("1", "1")
        print("query UserB address: org2 admin".center(50, "="))
        user_b_address = get_user_addr("2", "2")
        print("query UserC address: org3 admin".center(50, "="))
        user_c_address = get_user_addr("3", "3")
        print("query UserD address: org4 admin".center(50, "="))
        user_d_address = get_user_addr("4", "4")
        print("User ABCD address:", user_a_address, user_b_address, user_c_address, user_d_address)

        if gl.ENABLE_GAS == True:
            cmd = Command(sync_result=True)
            cmd.recharge_gas(user_a_address)
            cmd.recharge_gas(user_b_address)
            cmd.recharge_gas(user_c_address)
            cmd.recharge_gas(user_d_address)

        print("\n","1.rust asset 合约安装".center(50, "="))
        cd_asset = ContractDeal("rustsql", sync_result=True)
        result_erc = cd_asset.create("WASMER", "rust-sql-2.0.0.wasm",
                                     public_identity=f'{gl.ACCOUNT_TYPE}', sdk_config='sdk_config.yml',
                                     endorserKeys=f'{gl.ADMIN_KEY_FILE_PATHS}',endorserCerts=f'{gl.ADMIN_CRT_FILE_PATHS}',
                                     endorserOrgs=f'{gl.ADMIN_ORG_IDS}')
        asset_address = json.loads(result_erc).get("contract_result").get("result").get("address")
        print("rust asset 合约地址:",asset_address,"\n")


        print("2.执行合约-sql insert".center(50, "="))
        for i in range(1,10):
            cd_asset.invoke("sql_insert",sdk_config="sdk_config.yml",params="{{\"id\":\"{}\",\"name\":\"{}\",\"age\":\"{}\",\"id_card_no\":\"{}\"}}".format(str(i),"长安链chainmaker",str(i+10),"510623199202023323"))



        print("3.查询age为11的记录".center(50, "="))
        query_result=cd_asset.invoke("sql_query_by_id",sdk_config="sdk_config.yml",params="{{\"id\":\"{}\"}}".format(str(1)))
        query=base64.b64decode(json.loads(query_result).get("contract_result").get("result"))
        print("id:",str(base64.b64decode(json.loads(query).get("id")),encoding='utf-8'))
        print("name:",str(base64.b64decode(json.loads(query).get("name")),encoding='utf-8'))
        print("age:",str(base64.b64decode(json.loads(query).get("age")),encoding='utf-8'))
        print("id_card_no:",str(base64.b64decode(json.loads(query).get("id_card_no")),encoding='utf-8'),"\n")


        print("4.执行sql语: update name=长安链chainmaker_update where id=1".center(50, "="))
        cd_asset.invoke("sql_update",sdk_config="sdk_config.yml",params="{{\"id\":\"{}\",\"name\":\"{}\"}}".format(str(1),"长安链chainmaker_update"))



        print("5.查询id=1的name是不是更新成了长安链chainmaker_update".center(50, "="))
        name_update_result=cd_asset.get("sql_query_by_id",sdk_config="sdk_config.yml",params="{{\"id\":\"{}\"}}".format(str(1)))
        name_update=base64.b64decode(json.loads(name_update_result).get("contract_result").get("result"))
        print("id:",str(base64.b64decode(json.loads(name_update).get("id")),encoding='utf-8'))
        print("name:",str(base64.b64decode(json.loads(name_update).get("name")),encoding='utf-8'))
        print("age:",str(base64.b64decode(json.loads(name_update).get("age")),encoding='utf-8'))
        print("id_card_no:",str(base64.b64decode(json.loads(name_update).get("id_card_no")),encoding='utf-8'),"\n")


        print("6.范围查询 rang age 1~10".center(50, "="))
        range_age_result=cd_asset.invoke("sql_query_range_of_age",sdk_config="sdk_config.yml",params="{{\"min_age\":\"{}\",\"max_age\":\"{}\"}}".format(str(13),str(17)))
        range_age=str(base64.b64decode(json.loads(range_age_result).get("contract_result").get("result")),encoding='utf-8')
        parts_range_age=re.split('{|}',range_age)
        for part_range_age in parts_range_age:
            if part_range_age=='':
                continue
            part_range_age='{'+part_range_age+'}'
            print("id:",str(base64.b64decode(json.loads(part_range_age).get("id")),encoding='utf-8'))
            print("name:",str(base64.b64decode(json.loads(part_range_age).get("name")),encoding='utf-8'))
            print("age:",str(base64.b64decode(json.loads(part_range_age).get("age")),encoding='utf-8'))
            print("id_card_no:",str(base64.b64decode(json.loads(part_range_age).get("id_card_no")),encoding='utf-8'),"\n")


        print("7.执行合约-sql delete by id age=11".center(50, "="))
        cd_asset.invoke("sql_delete",sdk_config="sdk_config.yml",params="{{\"id\":\"{}\"}}".format(str(1)))


        print("8.再次查询 id age=11，应该查不到".center(50, "="))
        query_id_result=cd_asset.get("sql_query_by_id",sdk_config="sdk_config.yml",params="{{\"id\":\"{}\"}}".format(str(1)))
        query_id=str(base64.b64decode(json.loads(query_id_result).get("contract_result").get("result")),encoding='utf-8')
        print("再次查询 id age=11,结果为: ",query_id,"\n")
        self.assertEqual(query_id, '{}', "success")


        print("9.跨合约调用".center(50, "="))
        sql_cross_call_result=cd_asset.get("sql_cross_call",sdk_config="sdk_config.yml",params="{{\"contract_name\":\"{}\",\"min_age\":\"{}\",\"max_age\":\"{}\"}}".format("rustsql",str(16),str(19)))
        sql_cross_call=str(base64.b64decode(json.loads(sql_cross_call_result).get("contract_result").get("result")),encoding='utf-8')
        parts_range_age=re.split('{|}',sql_cross_call)
        for part_range_age in parts_range_age:
            if part_range_age=='':
                continue
            part_range_age='{'+part_range_age+'}'
            print("id:",str(base64.b64decode(json.loads(part_range_age).get("id")),encoding='utf-8'))
            print("name:",str(base64.b64decode(json.loads(part_range_age).get("name")),encoding='utf-8'))
            print("age:",str(base64.b64decode(json.loads(part_range_age).get("age")),encoding='utf-8'))
            print("id_card_no:",str(base64.b64decode(json.loads(part_range_age).get("id_card_no")),encoding='utf-8'),"\n")


        print("10.交易回退".center(50, "="))
        cd_asset.invoke("sql_insert",sdk_config="sdk_config.yml",params="{{\"id\":\"{}\",\"name\":\"{}\",\"age\":\"{}\",\"id_card_no\":\"{}\"}}".format(str(20),"长安链chainmaker",str(2000),"510623199202023323"))

        print("10.1 提交一笔执行会失败的交易".center(50, "="))
        cd_asset.invoke("sql_update_rollback_save_point",sdk_config="sdk_config.yml",params="{{\"id\":\"{}\",\"name\":\"{}\"}}".format(str(20),"chainmaker_save_point"))

        print("10.2 查询提交的失败交易有没有对上一笔产生影响".center(50, "="))
        query_id_result=cd_asset.get("sql_query_by_id",sdk_config="sdk_config.yml",params="{{\"id\":\"{}\"}}".format(str(20)))
        query_id=str(base64.b64decode(json.loads(query_id_result).get("contract_result").get("result")),encoding='utf-8')

        print("id:",str(base64.b64decode(json.loads(query_id).get("id")),encoding='utf-8'))
        print("name:",str(base64.b64decode(json.loads(query_id).get("name")),encoding='utf-8'))
        print("age:",str(base64.b64decode(json.loads(query_id).get("age")),encoding='utf-8'))
        print("id_card_no:",str(base64.b64decode(json.loads(query_id).get("id_card_no")),encoding='utf-8'),"\n")

        name=str(base64.b64decode(json.loads(query_id).get("name")),encoding='utf-8')
        self.assertEqual(name, '长安链chainmaker', "success")


        print("11.升级合约".center(50, "="))
        result_erc = cd_asset.upgrade("WASMER", "rust-sql-2.0.0.wasm",
                                      public_identity=f'{gl.ACCOUNT_TYPE}', sdk_config='sdk_config.yml',version="2.0.1",
                                      endorserKeys=f'{gl.ADMIN_KEY_FILE_PATHS}',endorserCerts=f'{gl.ADMIN_CRT_FILE_PATHS}',
                                      endorserOrgs=f'{gl.ADMIN_ORG_IDS}')
        asset_address = json.loads(result_erc).get("contract_result").get("result").get("address")
        print("rust asset 合约地址:",asset_address,"\n")


        print("12.升级合约后执行插入".center(50, "="))
        cd_asset.invoke("sql_insert",sdk_config="sdk_config.yml",params="{{\"id\":\"{}\",\"name\":\"{}\",\"age\":\"{}\",\"id_card_no\":\"{}\"}}".format(str(21),"长安链chainmaker",str(100000),"510623199202023323"))

        query_id_result=cd_asset.get("sql_query_by_id",sdk_config="sdk_config.yml",params="{{\"id\":\"{}\"}}".format(str(21)))
        query_id=str(base64.b64decode(json.loads(query_id_result).get("contract_result").get("result")),encoding='utf-8')

        print("id:",str(base64.b64decode(json.loads(query_id).get("id")),encoding='utf-8'))
        print("name:",str(base64.b64decode(json.loads(query_id).get("name")),encoding='utf-8'))
        print("age:",str(base64.b64decode(json.loads(query_id).get("age")),encoding='utf-8'))
        print("id_card_no:",str(base64.b64decode(json.loads(query_id).get("id_card_no")),encoding='utf-8'),"\n")

        age=str(base64.b64decode(json.loads(query_id).get("age")),encoding="utf-8")
        self.assertEqual(age,str(100000),"success")


        print("13.并发测试".center(50, "="))
        for i in range(500,600):
            cd_asset2 = ContractDeal("rustsql", sync_result=False)
            cd_asset2.invoke("sql_insert",sdk_config="sdk_config.yml",params="{{\"id\":\"{}\",\"name\":\"{}\",\"age\":\"{}\",\"id_card_no\":\"{}\"}}".format(str(i),"长安链chainmaker",str(i+10),"510623199202023323"))



        print("14.异常功能测试".center(50, "="))
        print("14.1 建表、索引、视图等DDL语句只能在合约安装init_contract 和合约升级upgrade中使用".center(50, "="))
        ddl_result=cd_asset.invoke("sql_execute_ddl",sdk_config="sdk_config.yml",params="{{\"id\":\"{}\",\"name\":\"{}\"}}".format(str(501),"长安链chainmaker"))
        ddl_message=json.loads(ddl_result).get("contract_result").get("message");
        b="符合预期" in ddl_message
        if b:
            print("result contains 符合预期 pass!!!\n")


        print("14.2 SQL中，禁止跨数据库操作，无需指定数据库名。比如select * from db.table 是禁止的； use db;是禁止的。".center(50, "="))
        forbidden_result=cd_asset.invoke("sql_dbname_table_name",sdk_config="sdk_config.yml",params="{{\"id\":\"{}\",\"name\":\"{}\"}}".format(str(501),"长安链chainmaker"))
        forbidden_message=json.loads(forbidden_result).get("contract_result").get("message");
        b="符合预期" in forbidden_message
        if b:
            print("result contains 符合预期 pass!!!\n")


        print("14.3 SQL中，禁止使用事务相关操作的语句，比如commit 、rollback等，事务由ChainMaker框架自动控制。".center(50, "="))
        tx_result=cd_asset.invoke("sql_execute_commit",sdk_config="sdk_config.yml",params="{{\"id\":\"{}\",\"name\":\"{}\"}}".format(str(501),"长安链chainmaker"))
        tx_message=json.loads(tx_result).get("contract_result").get("message");
        b="符合预期" in tx_message
        if b:
            print("result contains 符合预期 pass!!!\n")


        print("14.4 SQL中，禁止使用随机数、获得系统时间等不确定性函数，这些函数在不同节点产生的结果可能不一样，导致合约执行结果无法达成共识。".center(50, "="))
        random_key_result=cd_asset.invoke("sql_random_key",sdk_config="sdk_config.yml",params="{{\"id\":\"{}\",\"name\":\"{}\"}}".format(str(501),"长安链chainmaker"))
        random_key=json.loads(random_key_result).get("contract_result").get("message");
        b="forbidden sql keyword" in random_key
        if b:
            print("result contains forbidden sql keyword, pass!!!\n")

        random_str_result=cd_asset.invoke("sql_random_str",sdk_config="sdk_config.yml",params="{{\"id\":\"{}\",\"name\":\"{}\"}}".format(str(502),"长安链chainmaker"))
        random_str=str(base64.b64decode(json.loads(random_str_result).get("contract_result").get("result")),encoding='utf-8')
        self.assertEqual(random_str,"ok","success")
        print("result:",random_str, "pass !!!\n")

        random_query_str_result=cd_asset.get("sql_random_query_str",sdk_config="sdk_config.yml",params="{{\"id\":\"{}\",\"name\":\"{}\"}}".format(str(501),"长安链chainmaker"))
        random_query_str=str(base64.b64decode(json.loads(random_query_str_result).get("contract_result").get("result")),encoding='utf-8')
        self.assertEqual(random_query_str,"ok","success")
        print("result:",random_query_str, "pass !!!\n")

        print("14.5 SQL中，禁止多条SQL拼接成一个SQL字符串传入。".center(50, "="))
        multi_sql_result=cd_asset.invoke("sql_multi_sql",sdk_config="sdk_config.yml",params="{{\"id\":\"{}\",\"name\":\"{}\"}}".format(str(501),"长安链chainmaker"))
        multi_sql=json.loads(multi_sql_result).get("contract_result").get("message");
        b="符合预期" in multi_sql
        if b:
            print("result contains 符合预期 pass!!!\n")


        print("14.6 禁止建立、修改或删除表名为“state_infos”的表，这是系统自带的提供KV数据存储的表，用于存放PutState函数对应的数据。".center(50, "="))
        update_state_info_result=cd_asset.invoke("sql_update_state_info",sdk_config="sdk_config.yml",params="{{\"id\":\"{}\",\"name\":\"{}\"}}".format(str(501),"长安链chainmaker"))
        update_state_info=json.loads(update_state_info_result).get("contract_result").get("message");
        b="you can't change table state_infos" in update_state_info
        if b:
            print("result contains you can't change table state_infos","pass!!!\n")

        query_state_info_result=cd_asset.get("sql_query_state_info",sdk_config="sdk_config.yml",params="{{\"id\":\"{}\",\"name\":\"{}\"}}".format(str(501),"长安链chainmaker"))
        query_state_info=json.loads(query_state_info_result).get("contract_result").get("message");
        b="符合预期" in query_state_info
        if b:
            print("result contains 符合预期 pass!!!\n")



if __name__ == '__main__':
    unittest.main()