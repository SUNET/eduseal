import grpc
import time
from concurrent import futures

from eduseal.sealer.sealer import Sealer
from eduseal.models import PDFSignRequest
from eduseal.sealer.v1_sealer_pb2 import SealReply
import eduseal.sealer.v1_sealer_pb2_grpc as pb2_grpc

from eduseal.sealer.v1_sealer_pb2 import SealRequest, SealReply

class GRPCSealer(Sealer, pb2_grpc.SealerServicer):
    def __init__(self)->None:
        Sealer.__init__(self)

    def Seal(self, request:SealRequest, context) -> SealReply:
        self.logger.info(f"start seal document: {request.transaction_id}")

        reply = self.seal(request)

        return reply

if __name__ == "__main__":
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    pb2_grpc.add_SealerServicer_to_server(GRPCSealer(), server)
    #server.add_insecure_port("[::]:50051")
    server.add_secure_port("[::]:50051", grpc.ssl_server_credentials(
        [(open('/etc/ssl/private/sealer.crt').read(), open('/etc/ssl/private/sealer.key').read())]
    ))
    server.start()
    time.sleep(2)
    open('/tmp/healthcheck','w')
    server.wait_for_termination()