import { TType } from '../gen/api/proto/tournament_service/tournament_service_pb';
import { getEnumLabel } from '../utils/protobuf';

it('tests the enumToOptions buf stuff', () => {
  expect(getEnumLabel(TType, 2)).toBe('CHILD');
  expect(getEnumLabel(TType, 0)).toBe('STANDARD');
  expect(getEnumLabel(TType, -1)).toBe(undefined);
});
