import { proto3 } from '@bufbuild/protobuf';
import { TType } from '../gen/api/proto/tournament_service/tournament_service_pb';

it('tests the proto3 buf stuff', () => {
  expect(proto3.getEnumType(TType).findNumber(2)?.name).toBe('CHILD');
  expect(proto3.getEnumType(TType).findNumber(0)?.name).toBe('STANDARD');
  expect(proto3.getEnumType(TType).findNumber(-1)?.name).toBe(undefined);
});
