import grpc
import savt_pb2
import savt_pb2_grpc
from google.protobuf import empty_pb2


def main():
    # Connect to local gRPC service
    channel = grpc.insecure_channel('localhost:40001')
    client = savt_pb2_grpc.SavtIpcStub(channel)

    # 1. Echo
    try:
        resp = client.Echo(savt_pb2.EchoRequest(message="hello"))
        print("Echo:", resp)
    except Exception as e:
        print("Echo call failed:", e)

    # 2. GetStatus
    try:
        resp = client.GetStatus(savt_pb2.JobIdRequest(job_id="test-job-id"))
        print("GetStatus:", resp)
    except Exception as e:
        print("GetStatus call failed:", e)

    # 3. Start
    try:
        resp = client.Start(empty_pb2.Empty())
        print("Start:", resp)
    except Exception as e:
        print("Start call failed:", e)

    # 4. Kill
    try:
        resp = client.Kill(savt_pb2.JobIdRequest(job_id="test-job-id"))
        print("Kill:", resp)
    except Exception as e:
        print("Kill call failed:", e)

    # 5. ReadLog (streaming response, with timeout)
    print("ReadLog:")
    try:
        for log_entry in client.ReadLog(savt_pb2.JobIdRequest(job_id="test-job-id"), timeout=5):
            print(log_entry)
    except Exception as e:
        print("ReadLog call failed:", e)

    # 6. GetConfig
    try:
        resp = client.GetConfig(empty_pb2.Empty())
        print("GetConfig:", resp)
    except Exception as e:
        print("GetConfig call failed:", e)

    # 7. SetConfig (use config from GetConfig to avoid errors)
    try:
        config = client.GetConfig(empty_pb2.Empty())
        resp = client.SetConfig(config)
        print("SetConfig:", resp)
    except Exception as e:
        print("SetConfig call failed:", e)

    # 8. GetHistory
    try:
        resp = client.GetHistory(empty_pb2.Empty())
        print("GetHistory:", resp)
    except Exception as e:
        print("GetHistory call failed:", e)


if __name__ == "__main__":
    main() 