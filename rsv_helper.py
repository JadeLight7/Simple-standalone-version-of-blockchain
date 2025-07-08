# pip install web3
from web3 import Web3

# 初始化 Web3，使用 Infura 主网节点
w3 = Web3(Web3.HTTPProvider('https://mainnet.infura.io/v3/37817291b672499cb230df7589cf4b18'))

# 要查询的交易哈希（可以替换）
#txhash:0x8c0f0d8849ce6d01f135dd771c9761d967060d3bb13a1b6232b14dd58b064fa2
#v: 0
#r: 0x57107b76d3be4ceadbccd4571f4eca5f9cdd1d773582ca21469098ca41742fc1
#s: 0x3d9425a7ce5345a1778aa943af87f9ddba1dfe7f65825e5f50c0a6e88ec102
#From:0x95222290DD7278Aa3Ddd389Cc1E1d165CC4BAfe5

#txhash:0x4d5919b8f3dfd5e6756cdc0a412872fc89af9a7a9e1436d38c97da52427c47f0
#v: 0
#r: 0x52012529dde1b93192569e63a483f54f0b39b1d6c0a7d061bf0be6f2602ef5f1
#s: 0x45eaf65c58885525bc66477053718d9be4a6fee7be4cad08acdfdf1c99bab457
# From:0xd313777FA082515E7534Eff563519c7110821141
tx_hash = '0x8c0f0d8849ce6d01f135dd771c9761d967060d3bb13a1b6232b14dd58b064fa2'

# 判断连接是否成功
if not w3.is_connected():
    print("❌ 无法连接到以太坊节点")
    exit()

try:
    # 查询交易
    tx = w3.eth.get_transaction(tx_hash)

    print("=== 交易信息 ===")
    print("From    :", tx['from'])
    print("To      :", tx['to'] if tx['to'] else "[合约创建]")
    print("Nonce   :", tx['nonce'])
    print("Gas     :", tx['gas'])
    print("GasPrice:", w3.from_wei(tx['gasPrice'], 'gwei'), "GWei")
    print("Value   :", w3.from_wei(tx['value'], 'ether'), "ETH")

    print("\n=== 签名信息 ===")
    print("v:", tx["v"])
    print("r:", hex(int.from_bytes(tx["r"], byteorder='big')))
    print("s:", hex(int.from_bytes(tx["s"], byteorder='big')))




except Exception as e:
    print(f"❌ 查询交易失败: {e}")
