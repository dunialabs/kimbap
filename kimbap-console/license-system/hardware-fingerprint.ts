import { createHash } from 'crypto';
import * as os from 'os';
import { execSync } from 'child_process';

export interface HardwareInfo {
  platform: string;
  cpuCores: number;
  arch: string;
  hostname: string;
  macAddresses: string[];
  diskSerial?: string;
  motherboardSerial?: string;
}

export class HardwareFingerprint {
  private static getMacAddresses(): string[] {
    const networkInterfaces = os.networkInterfaces();
    const macs: string[] = [];
    
    for (const interfaceName of Object.keys(networkInterfaces)) {
      const interfaces = networkInterfaces[interfaceName];
      if (interfaces) {
        for (const iface of interfaces) {
          if (!iface.internal && iface.mac !== '00:00:00:00:00:00') {
            macs.push(iface.mac);
          }
        }
      }
    }
    
    return macs.sort();
  }

  private static getDiskSerial(): string | undefined {
    try {
      const platform = os.platform();
      let command: string;
      
      switch (platform) {
        case 'darwin':
          command = 'system_profiler SPSerialATADataType SPNVMeDataType SPUSBDataType 2>/dev/null | grep "Serial Number" | head -1 | awk \'{print $3}\'';
          break;
        case 'win32':
          command = 'wmic diskdrive get serialnumber /value | findstr SerialNumber';
          break;
        case 'linux':
          command = 'lsblk -o SERIAL -n | head -1';
          break;
        default:
          return undefined;
      }
      
      const result = execSync(command, { encoding: 'utf8' }).trim();
      return result || undefined;
    } catch (error) {
      console.error('Failed to get disk serial:', error);
      return undefined;
    }
  }

  private static getMotherboardSerial(): string | undefined {
    try {
      const platform = os.platform();
      let command: string;
      
      switch (platform) {
        case 'darwin':
          command = 'ioreg -l | grep IOPlatformSerialNumber | awk \'{print $4}\' | sed \'s/"//g\'';
          break;
        case 'win32':
          command = 'wmic baseboard get serialnumber /value | findstr SerialNumber';
          break;
        case 'linux':
          command = 'sudo dmidecode -s baseboard-serial-number 2>/dev/null';
          break;
        default:
          return undefined;
      }
      
      const result = execSync(command, { encoding: 'utf8' }).trim();
      return result || undefined;
    } catch (error) {
      console.error('Failed to get motherboard serial:', error);
      return undefined;
    }
  }

  public static getHardwareInfo(): HardwareInfo {
    return {
      platform: os.platform(),
      cpuCores: os.cpus().length,
      arch: os.arch(),
      hostname: os.hostname(),
      macAddresses: this.getMacAddresses(),
      diskSerial: this.getDiskSerial(),
      motherboardSerial: this.getMotherboardSerial()
    };
  }

  public static generateFingerprint(): string {
    const info = this.getHardwareInfo();
    
    const fingerprintData = {
      platform: info.platform,
      arch: info.arch,
      diskSerial: info.diskSerial || '',
      motherboardSerial: info.motherboardSerial || ''
    };
    
    const fingerprintString = JSON.stringify(fingerprintData);
    const hash = createHash('sha256').update(fingerprintString).digest('hex');
    
    return hash;
  }

  public static generateFingerprintWithSalt(salt: string): string {
    const baseFingerprint = this.generateFingerprint();
    const saltedFingerprint = `${baseFingerprint}:${salt}`;
    const hash = createHash('sha256').update(saltedFingerprint).digest('hex');
    
    return hash;
  }
}
