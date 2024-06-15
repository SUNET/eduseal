from kafka import KafkaConsumer

from eduseal.sealer.sealer import Sealer

class kafkaSigner(Sealer):
    def __init__(self, service_name: str = "signer_kafka"):
        self.service_name = service_name
        self.sign_queue= KafkaConsumer(self.config.sign_queue_name)
    


if __name__ == "__main__":
    sq = kafkaSigner()

    for msg in sq.sign_queue:
        print (msg)

    while True:
        sign_task = sq.sign_queue.wait()
        sign_data = sq.unmarshal(sign_task.data)
        sq.logger.info("Received signing task urn: %s transaction_id: %s", sign_task.urn, sign_data.transaction_id)
        signedPDF = sq.seal(in_data=sign_data)
        sq.logger.info("signedpdf: %s", signedPDF)

        s =  signedPDF.model_dump()
        add_signed_task = Task(data=signedPDF.model_dump())
        
        sq.add_signed_queue.enqueue(task=add_signed_task)