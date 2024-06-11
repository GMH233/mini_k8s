from prometheus_client import Gauge,Counter, start_http_server
import time
import random

# 定义和注册指标
random_value = Gauge('random_value', '随机数指标')


# 启动 HTTP 服务器，暴露 metrics 接口
start_http_server(32001)

while True:
    # 生成 0 到 100 的随机数，并设置到指标中
    random_value.set(random.randint(0, 100))

    # 等待 5 秒钟，再次进行收集
    time.sleep(1)