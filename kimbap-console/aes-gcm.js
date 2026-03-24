#!/usr/bin/env node
// AES-256-GCM CLI helper.
// Usage:
//   node aes-gcm.js enc <32-byte-key> "<plaintext>"
//   node aes-gcm.js dec <32-byte-key> '<json-from-encrypt>'
//   32-byte-key is a hex string, "Kimbap Console" md5 hash value. 88903baf62eb9d73f307c545198e5356
//   example: node aes-gcm.js enc 88903baf62eb9d73f307c545198e5356 '[{\"clientId\":\"YOUR_CLIENT_ID\",\"clientSecret\":\"YOUR_CLIENT_SECRET\"}]'
//   {"iv":"UufhjS9L2WfsoZdC","tag":"f0/QkCxf2Ua7HuZJY+f1LA==","ciphertext":"40U+9UuvxsicOFkKQkLZhoC4V5B+YffUnN/htbyCaQFw+Hgu/pLtFURFHCwUVj0Pn6vON5Wu0UJc/QXjlpVQa5Kzy5HBa/r4FrCB"}
//   node aes-gcm.js dec 88903baf62eb9d73f307c545198e5356 '{"iv":"UufhjS9L2WfsoZdC","tag":"f0/QkCxf2Ua7HuZJY+f1LA==","ciphertext":"40U+9UuvxsicOFkKQkLZhoC4V5B+YffUnN/htbyCaQFw+Hgu/pLtFURFHCwUVj0Pn6vON5Wu0UJc/QXjlpVQa5Kzy5HBa/r4FrCB"}'
//   [{"clientId":"YOUR_CLIENT_ID","clientSecret":"YOUR_CLIENT_SECRET"}]

const crypto = require('crypto');

const AAD = Buffer.from('Kimbap Consol', 'utf8');
const IV_LENGTH = 12; // 96-bit IV per NIST recommendation for GCM.

function exitWithUsage(msg) {
  if (msg) console.error(`Error: ${msg}`);
  console.error('Usage: node aes-gcm.js <enc|dec> <32-byte-key> <text|json>');
  process.exit(1);
}

function validateKey(keyStr) {
  const keyBuf = Buffer.from(keyStr, 'utf8');
  if (keyBuf.length !== 32) {
    exitWithUsage('Key must be exactly 32 bytes (AES-256).');
  }
  return keyBuf;
}

function encrypt(keyStr, plaintext) {
  const key = validateKey(keyStr);
  const iv = crypto.randomBytes(IV_LENGTH);
  const cipher = crypto.createCipheriv('aes-256-gcm', key, iv);
  cipher.setAAD(AAD, { plaintextLength: Buffer.byteLength(plaintext, 'utf8') });

  const ciphertext = Buffer.concat([cipher.update(plaintext, 'utf8'), cipher.final()]);
  const tag = cipher.getAuthTag();

  const payload = {
    iv: iv.toString('base64'),
    tag: tag.toString('base64'),
    ciphertext: ciphertext.toString('base64'),
  };

  console.log(JSON.stringify(payload));
}

function decrypt(keyStr, jsonStr) {
  const key = validateKey(keyStr);
  let payload;
  try {
    payload = JSON.parse(jsonStr);
  } catch (err) {
    exitWithUsage('Third argument must be valid JSON produced by encrypt.');
  }

  const { iv, tag, ciphertext } = payload || {};
  if (!iv || !tag || !ciphertext) {
    exitWithUsage('JSON must contain iv, tag, ciphertext fields.');
  }

  const ivBuf = Buffer.from(iv, 'base64');
  const tagBuf = Buffer.from(tag, 'base64');
  const ctBuf = Buffer.from(ciphertext, 'base64');

  const decipher = crypto.createDecipheriv('aes-256-gcm', key, ivBuf);
  decipher.setAAD(AAD, { plaintextLength: ctBuf.length });
  decipher.setAuthTag(tagBuf);

  let plaintext;
  try {
    plaintext = Buffer.concat([decipher.update(ctBuf), decipher.final()]).toString('utf8');
  } catch (err) {
    exitWithUsage('Decryption failed (bad key/iv/tag/ciphertext).');
  }

  console.log(plaintext);
}

function main() {
  const [, , mode, keyStr, data] = process.argv;
  if (!mode || !keyStr || data === undefined) {
    exitWithUsage();
  }

  if (mode === 'enc') {
    encrypt(keyStr, data);
  } else if (mode === 'dec') {
    decrypt(keyStr, data);
  } else {
    exitWithUsage('First argument must be "enc" or "dec".');
  }
}

main();
