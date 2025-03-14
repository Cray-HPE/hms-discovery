# Local Testing Environment for HMS-Discovery

## Bringup Test Containers
1. Startup the required Docker containers (do this in the root of this repo):

  ```bash
    docker-compose -f docker-compose.devel.yaml up
    ```
2. Load SLS file into SLS:
    
    ```bash
    curl -X POST -F sls_dump=@configs/slsTestConfig.json http://localhost:8376/v1/loadstate
    ```

## Manually Adding in Default Credentials
Exec into the vault container, and login to vault:
```bash
$ docker exec -it hms-discovery-vault-1 sh
/ # vault login hms
```
### REDS
Load in the credentials into `reds-creds`:
```bash
/ # echo '{
    "Cray": {
        "Username": "root",
        "Password": "cray"
    }
}' > vaultRedsDefaults.json
/ # vault kv put secret/reds-creds/defaults @vaultRedsDefaults.json
/ # vault kv put secret/reds-creds/switch_defaults SNMPUsername=user SNMPAuthPassword=snmpauth SNMPPrivPassword=snmppriv
```
### PDUs
Load in the default PDU Credentials
```bash
/ # vault kv put secret/pdu-creds/global/rts username=root password=rts
/ # vault kv put secret/pdu-creds/global/pdu username=root password=pdu
```

## Manually Populating the Ethernet Interfaces Table with unknown components
The MAC address for uan01 is `b42e993b7030`. So we need to add a blank ethernet interfaces entry into SMD
```json
{
    "MACAddress": "b4:2e:99:3b:70:30",
    "IPAddresses": [{"IPAddress": "10.252.0.34"}],
    "Description": "UAN"
}
```

```bash
curl -i -X POST -H "Content-Type: application/json" \
  -d '{"MACAddress":"b4:2e:99:3b:70:30", "IPAddresses":[{"IPAddress":"10.252.0.34"}], "Description": "UAN - Login" }' \
  http://localhost:27779/hsm/v2/Inventory/EthernetInterfaces
```

Lets add the PDU to the ethernet interfaces table, and the PDU's MAC address is `000a9c62202e`:
```json
{
    "MACAddress": "000a9c62202e",
    "IPAddresses": [{"IPAddress": "10.252.0.10"}],
    "Description": "PDU"
}
```

```bash
curl -i -X POST -H "Content-Type: application/json" \
  -d '{"MACAddress":"000a9c62202e", "IPAddresses":[{"IPAddress":"10.252.0.10"}], "Description": "PDU" }' \
  http://localhost:27779/hsm/v2/Inventory/EthernetInterfaces
```

## Running Discovery
The following shell script has the necessary environment variables set up and will launch discovery:

```bash
./runDiscovery.sh
```

## Resetting Env
The easiest way to reset the environment for another run of the discovery job is to clear out SMD, and re-adding the unknown components back into the ethernet interfaces table in SMD:
```bash
curl -X DELETE 'http://localhost:27779/hsm/v2/Inventory/EthernetInterfaces'
curl -X DELETE 'http://localhost:27779/hsm/v2/Inventory/RedfishEndpoints'
curl -X DELETE 'http://localhost:27779/hsm/v2/State/Components'

curl -i -X POST -H "Content-Type: application/json" \
  -d '{"MACAddress":"b42e993b7030", "IPAddresses":[{"IPAddress":"10.252.0.34"}], "Description": "UAN - Login" }' \
  http://localhost:27779/hsm/v2/Inventory/EthernetInterfaces

curl -i -X POST -H "Content-Type: application/json" \
  -d '{"MACAddress":"000a9c62202e", "IPAddresses":[{"IPAddress":"10.252.0.10"}], "Description": "PDU" }' \
  http://localhost:27779/hsm/v2/Inventory/EthernetInterfaces
```
