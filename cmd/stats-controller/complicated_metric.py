from prometheus_client import Counter, Gauge, Summary
from prometheus_client.core import CollectorRegistry
from prometheus_client.exposition import choose_encoder

import tornado
import tornado.ioloop
import tornado.web
import tornado.gen
from datetime import datetime
import random


class Monitor:
    random_value = Gauge('random_value', 'random number')

    def __init__(self):
        # 注册收集器
        self.collector_registry = CollectorRegistry(auto_describe=False)

        self.random_value.registry = self.collector_registry
        # 接口调用summary统计
        self.http_request_summary = Summary(name="http_server_requests",
                                   documentation="Num of request time summary",
                                   labelnames=("method", "code", "uri"),
                                   registry=self.collector_registry)
        

    # 获取/metrics结果
    def get_prometheus_metrics_info(self, handler):
        encoder, content_type = choose_encoder(handler.request.headers.get('accept'))
        handler.set_header("Content-Type", content_type)
        handler.write(encoder(self.collector_registry))

    # summary统计
    def set_prometheus_request_summary(self, handler):
        self.http_request_summary.labels(handler.request.method, handler.get_status(), handler.request.path).observe(handler.request.request_time())


global g_monitor
port = 32000


class PingHandler(tornado.web.RequestHandler):
    def get(self):
        print('INFO', datetime.now(), "/ping Get.")
        g_monitor.set_prometheus_request_summary(self)
        
        g_monitor.random_value.set(random.randint(0, 100))
        self.write("OK")


class MetricsHandler(tornado.web.RequestHandler):
    def get(self):
        print('INFO', datetime.now(), "/metrics Get.")
        g_monitor.set_prometheus_request_summary(self)

        g_monitor.random_value.set(random.randint(0, 100))
        random_metric = g_monitor.random_value.collect()[0].samples[0].value
		# 通过Metrics接口返回统计结果
        g_monitor.get_prometheus_metrics_info(self)
        self.write(f"\nrandom_value {random_metric}\n")
    

def make_app():
    return tornado.web.Application([
        (r"/ping?", PingHandler),
        (r"/metrics?", MetricsHandler),
    ])

if __name__ == "__main__":
    g_monitor = Monitor()
    app = make_app()
    app.listen(port)
    tornado.ioloop.IOLoop.current().start()