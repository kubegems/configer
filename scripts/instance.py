# -*- coding: utf-8 -*-

import nacos


# Both HTTP/HTTPS protocols are supported, if not set protocol prefix default is HTTP, and HTTPS with no ssl check(verify=False)
# "192.168.3.4:8848" or "https://192.168.3.4:443" or "http://192.168.3.4:8848,192.168.3.5:8848" or "https://192.168.3.4:443,https://192.168.3.5:443"

SERVER_ADDRESSES = "http://127.0.0.1:8848"
NAMESPACE = "59bd042678a226516a50e3badb0ac726b91cf393"
ak = "nacos"
sk = "nacos"

# client = nacos.NacosClient(SERVER_ADDRESSES, namespace=NAMESPACE)
client = nacos.NacosClient(SERVER_ADDRESSES, namespace=NAMESPACE, ak="{ak}", sk="{sk}")


service_name = "test"

client.add_naming_instance(service_name, "127.0.0.1", 8848)
client.add_naming_instance(service_name, "127.0.0.2", 8848)
client.add_naming_instance(service_name, "127.0.0.3", 8848)
