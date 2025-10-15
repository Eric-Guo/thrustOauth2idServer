import crypto from 'crypto';
import { promisify } from "node:util";

const pbkdf2 = promisify(crypto.pbkdf2);

// Works with Rails 7.1.5.2
// secretKeyBase is Rails.application.secret_key_base
async function decryptCookie(encryptedCookie: string, cookieName: string, secretKeyBase: string) {
  const cookie = decodeURIComponent(encryptedCookie);
  const [data, iv, authTag] = cookie.split("--").map(v => Buffer.from(v, 'base64'));
  if (!authTag || authTag.length !== 16) {
    throw new Error('InvalidMessage');
  }

  const salt = "authenticated encrypted cookie";
  const iterations = 1000;
  const keyLength = 32; // for AES-256
  const hashDigest = 'sha256'; // sha256 for Rails 7.2.2.2, sha1 for Rails 7.1.5.2
  // Generate secret key using PBKDF2
  const secret = await pbkdf2(secretKeyBase, salt, iterations, keyLength, hashDigest);
  // Setup cipher for decryption
  const decipher = crypto.createDecipheriv('aes-256-gcm', secret, iv);
  decipher.setAuthTag(authTag);
  decipher.setAAD(Buffer.from(''));
  try {
    // Perform decryption
    let decrypted = decipher.update(data, undefined, 'utf8');
    decrypted += decipher.final('utf8');
    const cookiePayload = JSON.parse(decrypted);
    const pur = cookiePayload["_rails"]['pur'];
    if (pur !== `cookie.${cookieName}`) {
      throw new Error('InvalidMessage');
    }
    const message = cookiePayload["_rails"]["message"];
    return JSON.parse(Buffer.from(message, 'base64').toString());
  } catch (err) {
    throw new Error('DecryptionFailed');
  }
}

function userIdFromDecryptedCookie(decryptedCookie: any) {
  return decryptedCookie["warden.user.user.key"][0][0];
}


const encryptedCookie = 'BxMj0jKfTfefrDppaGRrM4/IYGhNzt6bfRKxoZeWbmdYEGbrcp+5MGSdI3waNa+mLdo9GkIvkkKbew1XORZvs1wpMNrT1H1+akX6JnzLSdr3nJ4Ch+skQ9A8a9vC6JHE5RYGpaIKg+j3f+2sw6VqqgelkO0XyFfuy6W4JaOaDxIgLgrb7rg0BPcy8BRRPJKdhgL9FNt1mbl9RzE117dVc7tt3brR4gzVutO/h6PXcuYfiMetAKOPAEtZXaZIs3PVC/S8uyaqi+exjj0HqyJicFLgidwzuNRSX7asK/5UA++DntBj8Jd2v/Wx7ryyesp07Adv+yx/HdQTu3oTdx5yP98RueR1B1GiOPZmQwm3ePGPZfTQFS3sdiTvnAhRCzbRy7/yyyWmnyoL--5R6d/B6vwU11JVMW--c8v7EvGmB7ll4tdFKNgeHQ==';
const cookieName = '_oauth2id_session';
const secretKeyBase = '67f887242121133c0f7e8c4dba0c18aa8053071ded09181873093e93224546aa6ce592bf3312a5ccb8e0bb38242c149216ecef75cb4c0e9af2726703c769bcaf';

decryptCookie(encryptedCookie, cookieName, secretKeyBase)
  .then(userIdFromDecryptedCookie)
  .then(console.log);
