from concurrent import futures
import grpc
import time

from eduseal.validator.validator import Validator
import eduseal.validator.v1_validator_pb2_grpc as pb2_grpc
from eduseal.validator.v1_validator_pb2 import ValidateReply, ValidateRequest

class GRPCValidator(Validator, pb2_grpc.ValidatorServicer):
    def __init__(self)->None:
        Validator.__init__(self)

    def Validate(self, request:ValidateRequest, context) -> ValidateReply:
        self.logger.info(f"start validate document")

        reply = self.validate(request)
        self.logger.debug(f"validate reply: {reply}")

        return reply

if __name__ == "__main__":
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))

    pb2_grpc.add_ValidatorServicer_to_server(GRPCValidator(), server)

    server.add_insecure_port("[::]:50051")
    server.start()
    time.sleep(2)
    open('/tmp/healthcheck','w')
    server.wait_for_termination()