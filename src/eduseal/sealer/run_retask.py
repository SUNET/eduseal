from retask import Queue, Task, ConnectionError
from redis import Redis

from eduseal.sealer.sealer import Sealer

class RetaskSealer(Sealer):
    def __init__(self, service_name: str = "sealer_retask")->None:
        Sealer.__init__(self,service_name=service_name)
        self.service_name = service_name

        #r = Redis(host=self.config.redis.host, port=self.config.redis.port, password=self.config.redis.password, db=self.config.redis.db)

        #res = r.set(name="muuura", value="muuuraValue")
        #print("res", res)

        #exit(0)



        try:
            self.seal_queue = Queue(self.config.seal_queue_name, config=self.config.redis)
            self.seal_queue.connect()
        except ConnectionError:
            raise Exception("Can't connect to seal_queue")

        try:
            self.add_sealed_queue = Queue(self.config.add_sealed_queue_name, config=self.config.redis)
            self.add_sealed_queue.connect()
        except ConnectionError:
            raise Exception("Can't connect to sealed_queue")

if __name__ == "__main__":
    sq = RetaskSealer()

    while True:
        seal_task = sq.seal_queue.wait()
        sign_data = sq.unmarshal(seal_task.data)
        sq.logger.info("Received sealing task urn: %s transaction_id: %s", seal_task.urn, sign_data.transaction_id)
        sealedPDF = sq.seal(in_data=sign_data)
        sq.logger.info("sealed pdf: %s", sealedPDF)

        s =  sealedPDF.model_dump()
        add_signed_task = Task(data=sealedPDF.model_dump())
        
        sq.add_sealed_queue.enqueue(task=add_signed_task)