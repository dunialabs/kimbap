import { handleProtocol10001 } from './protocol-10001';
import { handleProtocol10002 } from './protocol-10002';
import { handleProtocol10003 } from './protocol-10003';
import { handleProtocol10004 } from './protocol-10004';
import { handleProtocol10005 } from './protocol-10005';
import { handleProtocol10006 } from './protocol-10006';
import { handleProtocol10007 } from './protocol-10007';
import { handleProtocol10008 } from './protocol-10008';
import { handleProtocol10009 } from './protocol-10009';
import { handleProtocol10010 } from './protocol-10010';
import { handleProtocol10011 } from './protocol-10011';
import { handleProtocol10012 } from './protocol-10012';
import { handleProtocol10013 } from './protocol-10013';
import { handleProtocol10014 } from './protocol-10014';
import { handleProtocol10015 } from './protocol-10015';
import { handleProtocol10016 } from './protocol-10016';
import { handleProtocol10017 } from './protocol-10017';
import { handleProtocol10018 } from './protocol-10018';
import { handleProtocol10019 } from './protocol-10019';
import { handleProtocol10020 } from './protocol-10020';
import { handleProtocol10021 } from './protocol-10021';
import { handleProtocol10022 } from './protocol-10022';
import { handleProtocol10023 } from './protocol-10023';
import { handleProtocol10024 } from './protocol-10024';
import { handleProtocol10025 } from './protocol-10025';
import { handleProtocol10040 } from './protocol-10040';
import { handleProtocol10041 } from './protocol-10041';
import { handleProtocol10042 } from './protocol-10042';
import { handleProtocol10043 } from './protocol-10043';
import { handleProtocol10050 } from './protocol-10050';
import { handleProtocol10051 } from './protocol-10051';
import { handleProtocol10052 } from './protocol-10052';
import { handleProtocol10053 } from './protocol-10053';
import { handleProtocol10054 } from './protocol-10054';
import { handleProtocol10055 } from './protocol-10055';
import { handleProtocol10056 } from './protocol-10056';
import { handleProtocol10057 } from './protocol-10057';
import { handleProtocol10058 } from './protocol-10058';
import { handleProtocol10059 } from './protocol-10059';
import { handleProtocol10060 } from './protocol-10060';
import { handleProtocol10061 } from './protocol-10061';
import { handleProtocol10062 } from './protocol-10062';
import { handleProtocol10063 } from './protocol-10063';
import { handleProtocol10064 } from './protocol-10064';
import { handleProtocol10065 } from './protocol-10065';
import { handleProtocol10066 } from './protocol-10066';
import { handleProtocol10067 } from './protocol-10067';
import { handleProtocol20001 } from './protocol-20001';
import { handleProtocol20002 } from './protocol-20002';
import { handleProtocol20003 } from './protocol-20003';
import { handleProtocol20004 } from './protocol-20004';
import { handleProtocol20005 } from './protocol-20005';
import { handleProtocol20006 } from './protocol-20006';
import { handleProtocol20007 } from './protocol-20007';
import { handleProtocol20008 } from './protocol-20008';
import { handleProtocol20009 } from './protocol-20009';
import { handleProtocol20010 } from './protocol-20010';
import { handleProtocol21001 } from './protocol-21001';
import { handleProtocol21002 } from './protocol-21002';
import { handleProtocol21003 } from './protocol-21003';
import { handleProtocol21004 } from './protocol-21004';
import { handleProtocol21005 } from './protocol-21005';
import { handleProtocol21006 } from './protocol-21006';
import { handleProtocol21007 } from './protocol-21007';
import { handleProtocol21008 } from './protocol-21008';
import { handleProtocol21009 } from './protocol-21009';
import { handleProtocol21010 } from './protocol-21010';
import { handleProtocol21011 } from './protocol-21011';
import { handleProtocol22001 } from './protocol-22001';
import { handleProtocol22002 } from './protocol-22002';
import { handleProtocol22003 } from './protocol-22003';
import { handleProtocol22004 } from './protocol-22004';
import { handleProtocol23001 } from './protocol-23001';
import { handleProtocol23002 } from './protocol-23002';
import { handleProtocol23004 } from './protocol-23004';
import { handleProtocol23005 } from './protocol-23005';
import { ApiError, ErrorCode } from '@/lib/error-codes';

export type ProtocolHandler = (body: any) => Promise<any>;

// Protocol handler registry
export const protocolHandlers: Record<number, ProtocolHandler> = {
  10001: handleProtocol10001,
  10002: handleProtocol10002,
  10003: handleProtocol10003,
  10004: handleProtocol10004,
  10005: handleProtocol10005,
  10006: handleProtocol10006,
  10007: handleProtocol10007,
  10008: handleProtocol10008,
  10009: handleProtocol10009,
  10010: handleProtocol10010,
  10011: handleProtocol10011,
  10012: handleProtocol10012,
  10013: handleProtocol10013,
  10014: handleProtocol10014,
  10015: handleProtocol10015,
  10016: handleProtocol10016,
  10017: handleProtocol10017,
  10018: handleProtocol10018,
  10019: handleProtocol10019,
  10020: handleProtocol10020,
  10021: handleProtocol10021,
  10022: handleProtocol10022,
  10023: handleProtocol10023,
  10024: handleProtocol10024,
  10025: handleProtocol10025,
  // Skills Management APIs (10040-10043)
  10040: handleProtocol10040,
  10041: handleProtocol10041,
  10042: handleProtocol10042,
  10043: handleProtocol10043,
  // Tool Usage Statistics APIs (20001-20010)
  20001: handleProtocol20001,
  20002: handleProtocol20002,
  20003: handleProtocol20003,
  20004: handleProtocol20004,
  20005: handleProtocol20005,
  20006: handleProtocol20006,
  20007: handleProtocol20007,
  20008: handleProtocol20008,
  20009: handleProtocol20009,
  20010: handleProtocol20010,
  // Access Token Usage Statistics APIs (21001-21010)
  21001: handleProtocol21001,
  21002: handleProtocol21002,
  21003: handleProtocol21003,
  21004: handleProtocol21004,
  21005: handleProtocol21005,
  21006: handleProtocol21006,
  21007: handleProtocol21007,
  21008: handleProtocol21008,
  21009: handleProtocol21009,
  21010: handleProtocol21010,
  21011: handleProtocol21011,
  // Usage Overview Statistics APIs (22001-22010)
  22001: handleProtocol22001,
  22002: handleProtocol22002,
  22003: handleProtocol22003,
  22004: handleProtocol22004,
  // Logs Management APIs (23001-23010)
  23001: handleProtocol23001,
  23002: handleProtocol23002,
  23004: handleProtocol23004,
  23005: handleProtocol23005,
  // Content-Aware Policy & Approval APIs (10050-10058)
  10050: handleProtocol10050,
  10051: handleProtocol10051,
  10052: handleProtocol10052,
  10053: handleProtocol10053,
  10054: handleProtocol10054,
  10055: handleProtocol10055,
  10056: handleProtocol10056,
  10057: handleProtocol10057,
  10058: handleProtocol10058,
  10059: handleProtocol10059,
  10060: handleProtocol10060,
  10061: handleProtocol10061,
  10062: handleProtocol10062,
  10063: handleProtocol10063,
  10064: handleProtocol10064,
  10065: handleProtocol10065,
  10066: handleProtocol10066,
  10067: handleProtocol10067,
  // Add more protocol handlers here as needed
  // etc.
};

export function getProtocolHandler(cmdId: number): ProtocolHandler | undefined {
  return protocolHandlers[cmdId];
}
