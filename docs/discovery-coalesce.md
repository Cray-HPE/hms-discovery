# Plan to coalesce discovery services in Shasta CSM 1.2

## Existing Responsibilities
### MEDS
- Generate Mountain/Hill Ethernet Interfaces for all possible mountain BMCs in the system
    - Does a patch to an Ethernet Interface if it already exists in HSM. Allows for the updating of the endpoints.
- Add Mountain/Hill RF endpoints to HSM as they become available on the network
    - Once the hostname of Mountain/Hill BMC starts to resolve and a 200 status code is returned when pinging the Redfish root, MEDS considers the BMC to be alive and will then add it to HSM.
    - Setup NTP/Rsyslog/SSH Keys on the Mountain/Hill BMCs.
- Load default Mountain/Hill BMC credentials into Vault.

### REDS
- Add RedfishEndpoints for the Management NCN BMCs to HSM. A periodic query to SLS is performed to determine the NCNs in the system.
    - Since the NCN BMCs have a static IP address and do not DHCP, the normal river discovery flow does not work. So they must be explicitly added.
- Setup NTP/Rsyslog/SSH Keys on the Colorado Slingshot switches.
- Load default River BMC and Switch SNMP credentials into Vault.

### Discovery Job
- Query HSM for MACs with unknown components. Determine the port that the MAC is associated with by using SNMP to query leaf switches in the system, and lookup in SLS to determine the identity of the device connected to that port. Then finally add the BMC as a RedfishEndpoint to HSM.
    - For River (ServerTech) PDUs it will add an entry into vault for RTS to handle.

- Turn on Slots for Mountain hardware. 
    1. Query HSM for Chassis, ComputeModule and RouterModules components
    2. Use CAPMC to get power status for these components
    3. Use CAPMC to power on any slots that are off.


## Changes
The existing useful functionality from REDS, MEDS, and the HMS Discovery Job will be coalesced into a single discovery service.

1. MEDS and REDS are no more.
2. Use a single vault loader to load the default credentials for Mountain and River BMCs alongside the default SNMP Switch credentials
3. Use a MAC based discovery process for Mountain BMCs in a similar fashion to River. Since the MAC address for a Mountain BMC is algorithmic the identity if the BMC can be easily identified.
    - This will remove the need for MEDS to continuously ping every possible Mountain/Hill BMC in the system, as the was the mechanism that was used to determine when redfish endpoints where alive.
4. For the BMCs that support it, set NTP/Rsyslog/SSHKeys on the BMCs.
    - Same code to handled all BMCs, instead of some of the responsibilities living in REDS and the reset in MEDS.
    - Removes the need for REDS to continuously trying to ping Columbia switches
5. Convert the Mountain discovery script used by the discovery job into Golang. Remove the dependency on Python.

## Additions
1. The Periodic checking of NTP/RSYSLOG/SSHKeys, and ensuring that they are configured correctly. If they are not (after hardware replacement perhaps or manual changes) reconfigure the BMC's.
    - Setting up and verifying that Redfish subscriptions still exist is the responsibility of the hms-collector.
    - The polling of leaf switches can be done in parallel. 
2. Monitor all redfish endpoints and verify that they have the correct configuration after they went through discovery
    - Verify that NTP/RSYSLOG/SSHKeys are setup correctly, and fix them if they are not right. Such as after hardware is replaced.
3. A robust way to determine when a BMC have actually been hardware swaps vs instabilities in the network.
    > Needs to be done in an way that does not hammer HSM with unneeded discovery requests.
    - Would allow automatic rediscovery of hardware after hardware work.

## Questions
1. What are the advantages of moving the discovery job into an periodic task into a service?
    - No longer have to rely on k8s to schedule tasks. If the tasks does take longer then that 3 minutes allowed the job won't be killed
        - The discovery service could be constantly query SNMP on the switches in the background, so it always has up to date information.
        - Then the MAC discovery periodic task can run on a faster rate, and can react to changes into hardware faster.
            - IE the time between: BMC DHCPs with KEA -> KEA Updates HSM ethernet interfaces -> discovery job runs every three minutes -> discovery identifies the BMC -> Unbound/PowerDNS gets updated with new host information -> BMC is discovered in HSM.
2. What are the disadvantages of moving the discovery job into an periodic task into a service?
    - The discovery job gets a fresh state to run with every time it runs, so any presentient bugs/errors are cleared automatically
    - It is a simple (and documented command to suspend the discovery of new things in the system)
        - Instead of suspending the cronjob the, the service can be scaled down.
