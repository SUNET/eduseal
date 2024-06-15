import grpc
from concurrent import futures

from eduseal.sealer.sealer import Sealer
from eduseal.models import PDFSignRequest
from eduseal.sealer.v1_sealer_pb2 import SealReply
import eduseal.sealer.v1_sealer_pb2_grpc as pb2_grpc


class GRPCSealer(Sealer, pb2_grpc.SealerServicer):
    def __init__(self, service_name: str = "sealer_grpc")->None:
        Sealer.__init__(self, service_name=service_name)
        self.service_name = service_name

    def Seal(self, request, context):
        self.logger.info(f"start seal document: {request.transaction_id}")

        to_be_sealed = PDFSignRequest(transaction_id=request.transaction_id, base64_data=request.pdf) 

        sealed_reply = self.seal(in_data=to_be_sealed)

        #self.logger.info(f"GRPC sealed document: {sealed_pdf.transaction_id}")
        #self.logger.info(f"GRPC sealed pdf: {sealed_pdf.base64_data}")

        return SealReply(
            transaction_id=sealed_reply.transaction_id, 
            pdf=sealed_reply.base64_data, 
            error=sealed_reply.error, 
            create_ts=sealed_reply.create_ts,
            )

if __name__ == "__main__":
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    pb2_grpc.add_SealerServicer_to_server(GRPCSealer(), server)
    server.add_insecure_port("[::]:50051")
    server.start()
    server.wait_for_termination()