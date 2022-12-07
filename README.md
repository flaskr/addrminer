# create3 address miner
The `addrminer` program generates salts for use in `create3` factories to create vanity addresses. 

Reference implementations of `create3` can be found in these repositories:
* https://github.com/transmissions11/solmate/blob/main/src/utils/CREATE3.sol
* https://github.com/0xSequence/create3/blob/master/contracts/Create3.sol

# Usage
```bash
 % go run . --help
  -factoryAddress string
        Address of the create3 factory being used
  -minOccur prefix
        minimum number of times the prefix must have appeared (default 1)
  -miningSalt int
        Salt to use for starting to search for salts (default 1)
  -prefix string
        hexstring prefix that you want in the address. Even number of characters. (default "00")
  -step int
        Step of increment for each try (default 1)
```

Found salts and addresses will be appended to `addresses.csv` file in the project folder.
