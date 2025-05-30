// @generated by protoc-gen-es v2.5.1 with parameter "target=ts"
// @generated from file proto/pair_service/pair_service.proto (package pair_service, syntax proto3)
/* eslint-disable */

import type { GenFile, GenService } from "@bufbuild/protobuf/codegenv2";
import { fileDesc, serviceDesc } from "@bufbuild/protobuf/codegenv2";
import type { PairRequestSchema, PairResponseSchema } from "../ipc/pair_pb";
import { file_proto_ipc_pair } from "../ipc/pair_pb";

/**
 * Describes the file proto/pair_service/pair_service.proto.
 */
export const file_proto_pair_service_pair_service: GenFile = /*@__PURE__*/
  fileDesc("CiVwcm90by9wYWlyX3NlcnZpY2UvcGFpcl9zZXJ2aWNlLnByb3RvEgxwYWlyX3NlcnZpY2UyRwoLUGFpclNlcnZpY2USOAoRSGFuZGxlUGFpclJlcXVlc3QSEC5pcGMuUGFpclJlcXVlc3QaES5pcGMuUGFpclJlc3BvbnNlQqoBChBjb20ucGFpcl9zZXJ2aWNlQhBQYWlyU2VydmljZVByb3RvUAFaOGdpdGh1Yi5jb20vd29vZ2xlcy1pby9saXdvcmRzL3JwYy9hcGkvcHJvdG8vcGFpcl9zZXJ2aWNlogIDUFhYqgILUGFpclNlcnZpY2XKAgtQYWlyU2VydmljZeICF1BhaXJTZXJ2aWNlXEdQQk1ldGFkYXRh6gILUGFpclNlcnZpY2ViBnByb3RvMw", [file_proto_ipc_pair]);

/**
 * @generated from service pair_service.PairService
 */
export const PairService: GenService<{
  /**
   * @generated from rpc pair_service.PairService.HandlePairRequest
   */
  handlePairRequest: {
    methodKind: "unary";
    input: typeof PairRequestSchema;
    output: typeof PairResponseSchema;
  },
}> = /*@__PURE__*/
  serviceDesc(file_proto_pair_service_pair_service, 0);

