from retask import Queue

from eduseal.validator.validator import Validator

class RetaskValidator(Validator):
    def __init__(self, service_name: str = "validator_retask")->None:
        Validator.__init__(self, service_name=service_name)
        self.service_name = service_name

        self.validate_queue = Queue(
            name=self.config.validate_queue_name, 
            config=self.config.redis,
        )
        self.validate_queue.connect()

if __name__ == "__main__":
    vq = RetaskValidator()

    while True:
        validate_task = vq.validate_queue.wait()
        vq.logger.info("Received validate task urn: %s", validate_task.urn)
        if isinstance(validate_task, bool):
            vq.logger.error("No task received, timeout!")
            exit(0)

        validate_data = vq.unmarshal(validate_task.data)

        validated_pdf = vq.validate(in_data=validate_data)

        vq.validate_queue.send(task=validate_task, result=validated_pdf.model_dump())