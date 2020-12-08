# Local Testing Environment for HMS-Discovery

## Bringup Test Containers
1. Startup the required Docker containers (do this in the root of this repo):
  - `$ docker-compose -f docker-compose.devel.yaml up`
2. Load SLS file into SLS:
  - `$ curl -X POST -F sls_dump=@configs/slsTestConfig.json http://localhost:8376/v1/loadstate`

## Manually Adding in Default Credentials
Exec into the vault container, and login to vault:
```
$ docker exec -it hms-discovery_vault_1 sh
/ # vault login hms
```
### REDS
Load in the credentials into `reds-creds`:
```
/ # echo '{
    "Cray": {
        "Username": "root",
        "Password": "cray"
    }
}' > vaultRedsDefaults.json
/ # vault kv put reds-creds/defaults @vaultRedsDefaults.json
/ # vault kv put reds-creds/switch_defaults SNMPUsername=user SNMPAuthPassword=snmpauth SNMPPrivPassword=snmppriv
```
### PDUs
Load in the default PDU Credentials
```
/ # vault kv put pdu-creds/global/rts username=root password=rts
/ # vault kv put pdu-creds/global/pdu username=root password=pdu
```

## Manually Populating the Ethernet Interfaces Table with unknown components
The MAC address for uan01 is `b42e993b7030`. So we need to add a blank ethernet interfaces entry into SMD
```json
{
    "MACAddress": "b4:2e:99:3b:70:30",
    "IPAddress": "10.252.0.34",
    "Description": "UAN"
}
```

```
$ curl -i -X POST -H "Content-Type: application/json" \
  -d '{"MACAddress":"b4:2e:99:3b:70:30", "IPAddress":"10.252.0.34", "Description": "UAN - Login" }' \
  http://localhost:27779/hsm/v1/Inventory/EthernetInterfaces
```

Lets add the PDU to the ethernet interfaces table, and the PDU's MAC address is `000a9c62202e`:
```json
{
    "MACAddress": "000a9c62202e",
    "IPAddress": "10.252.0.10",
    "Description": "PDU"
}
```

```
curl -i -X POST -H "Content-Type: application/json" \
  -d '{"MACAddress":"000a9c62202e", "IPAddress":"10.252.0.10", "Description": "PDU" }' \
  http://localhost:27779/hsm/v1/Inventory/EthernetInterfaces
```

## Running Discovery
The following environment variables are required to be set for Discovery to run properly:
```
SLS_URL=http://localhost:8376
HSM_URL=http://localhost:27779
CRAY_VAULT_AUTH_PATH=auth/token/create
CRAY_VAULT_ROLE_FILE=configs/namespace
CRAY_VAULT_JWT_FILE=configs/token
VAULT_ADDR=http://localhost:8200
VAULT_TOKEN=hms
SNMP_MODE=MOCK
DISCOVER_MOUNTAIN=false
DISCOVER_RIVER=true
LOG_LEVEL=DEBUG"
```

## Resetting Env
The easiest way to reset the environment for another run of the discovery job is to clear out SMD, and re-adding the unknown components back into the ethernet interfaces table in SMD:
``` 
curl -X DELETE 'http://localhost:27779/hsm/v1/Inventory/EthernetInterfaces'
curl -X DELETE 'http://localhost:27779/hsm/v1/Inventory/RedfishEndpoints'
curl -X DELETE 'http://localhost:27779/hsm/v1/State/Components'

curl -i -X POST -H "Content-Type: application/json" \
  -d '{"MACAddress":"b42e993b7030", "IPAddress":"10.252.0.34", "Description": "UAN - Login" }' \
  http://localhost:27779/hsm/v1/Inventory/EthernetInterfaces

curl -i -X POST -H "Content-Type: application/json" \
  -d '{"MACAddress":"000a9c62202e", "IPAddress":"10.252.0.10", "Description": "PDU" }' \
  http://localhost:27779/hsm/v1/Inventory/EthernetInterfaces
```