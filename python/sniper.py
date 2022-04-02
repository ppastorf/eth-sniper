import time
import json
from attributedict.collections import AttributeDict
import requests
from web3 import Web3, gas_strategies

network_address = "https://bsc-dataseed.binance.org/"
web3 = Web3(Web3.HTTPProvider(network_address))


class Wallet():
  def __init__(self, public_key, private_key):
    self.public_key  = public_key
    self.private_key = private_key

  def address(self):
    return self.public_key

  def getBalance(self, unit='ether'):
    wei = web3.eth.get_balance(self.public_key)
    return web3.fromWei(wei, unit)

  def sign_transaction(self, transaction):
    return web3.eth.account.sign_transaction(transaction, private_key=self.private_key)


class SwapTransaction():
  def __init__(self, router_address, router_abi, from_address, from_token, to_token, amountIn, expiration, gas=None,
                gas_strategy=None, max_fee_per_gas=None, max_priority_fee_per_gas=None):
    self.contract = web3.eth.contract(address=router_address, abi=router_abi)
    self.from_address = from_address
    self.from_token = web3.toChecksumAddress(from_token)
    self.to_token = web3.toChecksumAddress(to_token)
    self.amountIn = amountIn
    self.gas = 21000 if gas==None else gas
    self.gas_strategy = gas_strategy
    self.expiration = expiration
    self.max_fee_per_gas = max_fee_per_gas
    self.max_priority_fee_per_gas = max_priority_fee_per_gas

  def build(self):
    self.expiration_time = int(time.time()) + self.expiration

    # https://docs.pancakeswap.finance/code/smart-contracts/pancakeswap-exchange/router-v2#swapexactethfortokens
    func = self.contract.functions.swapExactETHForTokens(
      0,
      [self.from_token, self.to_token],
      self.from_address,
      self.expiration_time
    )
    
    tx = {
      'from': self.from_address,
      'value': self.amountIn,
      'gas': 0,
      'gasPrice': estimate_gas_price(strategy=self.gas_strategy),
      'nonce': web3.eth.get_transaction_count(self.from_address)
    }

    try:
      est_gas = func.estimateGas(tx)
      print(f"estimated gas limit (contract): {est_gas}")
      gas_limit = int(est_gas * 1.1)
      print(f"Estimated gas limit: {gas_limit}")
      tx['gas'] = gas_limit
    except Exception as err:
      print("Cannot estimate Gas limit: ", err)

    try:
      print(f"estimated gas limit (web3_rpc): {gas_strategies.rpc.rpc_gas_price_strategy(web3, transaction_params=tx)}")
    except Exception as err:
      print("cannot estimate gas: ", err)
    try:
      max_wait_seconds = 60
      probability = 99
      print(f"estimated gas limit (web3_time_based): {gas_strategies.time_based.construct_time_based_gas_price_strategy(max_wait_seconds, sample_size=120, probability=probability, weighted=False)}")
    except Exception as err:
      print("cannot estimate gas: ", err)

    return func.buildTransaction(tx)


def estimate_gas_price(strategy='safe'):
  estimates = get_estimate_gas_bsc()
  boost = 0.01 # gwei

  if strategy.lower() == 'proposed':
    gas_price = estimates.ProposeGasPrice
  elif strategy.lower() == 'fast':
    gas_price = estimates.FastGasPrice
  else: gas_price = estimates.SafeGasPrice

  return web3.toWei(float(gas_price)+boost, 'gwei')


def get_estimate_gas_bsc():
  url = 'https://api.bscscan.com/api?module=gastracker&action=gasoracle'
  res = requests.get(url)
  body = AttributeDict(json.loads(res.content))

  if body['message'] != 'OK' or res.status_code != 200:
      raise Exception(f"Failed to make request to {url}: {body}")

  return body.result


if __name__ == '__main__':
  if not web3.isConnected():
    print(f"Cannot connect to network at {network_address}")
    exit(1)

  wallet = Wallet(
    public_key='0xeaF241192d6f2DD14E546d76028C50d714e1e2CB',
    private_key='af44b8442594fd667f19779f79fc92e5f99aa8f150a3c0e171a942e33d9b8c08'
  )

  print('Current balance: ', wallet.getBalance(unit='ether'))

  pancakeswap_router_address = "0x10ED43C718714eb63d5aA57B78B54704E256024E"
  pancakeswap_router_abi = open('pancakeABI', 'r').read().replace('\n', '')

  tx = SwapTransaction(
    router_address=pancakeswap_router_address,
    router_abi=pancakeswap_router_abi,
    from_address=wallet.address(),
    from_token='0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c', # WBNB
    to_token='0x0E09FaBB73Bd3Ade0a17ECC321fD13a19e81cE82', # CAKE
    amountIn=web3.toWei(0.0002, 'ether'), # ether = WBNB
    gas_strategy='safe',
    expiration=60*5
  )

  built_tx = tx.build()
  print('tx: ', built_tx)

  signed_tx = wallet.sign_transaction(built_tx)

  sent_tx = web3.eth.send_raw_transaction(signed_tx.rawTransaction)
  print('sent transaction', web3.toHex(sent_tx))
  
  txn_receipt = None
  while txn_receipt is None and (time.time() < tx.expiration_time):
    try:
      txn_receipt = web3.eth.getTransactionReceipt(sent_tx)
      if txn_receipt is not None: 
          print(txn_receipt)
          break
      time.sleep(10)
    except Exception as err:
      print(err)
