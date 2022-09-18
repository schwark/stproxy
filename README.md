# SmartThings Edge Proxy 

This is a proxy needed to access web based resources from SmartThings Edge Drivers. This is locked down to only certain servers via a config.json file that needs to be in the same directory as the executable. The config file can also specify the port that the proxy runs on.


1. Identify your OS - one of `macos  linux  windows`

2. Identify your CPU architecture - one of `386 amd64 arm arm64`. This can be a little complicated

```
uname -m
```

if the output is.. 

```
output          architecture            examples
------          ------------            --------
386             386                     really old windows or linux PCs
x86_64          amd64                   most modern windows machines and most modern linux PCs 
                                        and modern Macs except M1/M2 based Macs
amd64           amd64                   most modern windows machines and most modern linux PCs 
                                        and modern Macs except M1/M2 based Macs
arm64           arm64                   Only Raspberry Pi 3 and above, and M1/M2 macs
armv6           arm                     Most Raspberry Pis below v3
armv7           arm                     Most Raspberry Pis below v3
armv8           arm64                   Only Raspberry Pi 4 and above
```

2. Download the appropriate [release](https://github.com/schwark/stproxy/releases/latest)

3. Download the config [file] (https://raw.githubusercontent.com/schwark/stproxy/main/config.json)

4. Copy both files to any directory and Run it and specify directory of config.json file if there is an error without it

```
stproxy-<os>-<arch> -d <config-directory>
```


