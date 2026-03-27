import { handleProtocol10002 } from './protocol-10002';
import { handleProtocol10015 } from './protocol-10015';
import { handleProtocol10023 } from './protocol-10023';
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

export type ProtocolHandler = (body: any) => Promise<any>;

export const protocolHandlers: Record<number, ProtocolHandler> = {
  10002: handleProtocol10002,
  10015: handleProtocol10015,
  10023: handleProtocol10023,
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
  22001: handleProtocol22001,
  22002: handleProtocol22002,
  22003: handleProtocol22003,
  22004: handleProtocol22004,
  23001: handleProtocol23001,
  23002: handleProtocol23002,
  23004: handleProtocol23004,
  23005: handleProtocol23005,
};

export function getProtocolHandler(cmdId: number): ProtocolHandler | undefined {
  return protocolHandlers[cmdId];
}
