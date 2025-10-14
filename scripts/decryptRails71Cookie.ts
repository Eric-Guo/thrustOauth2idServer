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
  const hashDigest = 'sha1'; // Rails 7.1 derives cookie keys with SHA1 PBKDF2
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

decryptCookie(encryptedCookie, cookieName, secretKeyBase)
  .then(userIdFromDecryptedCookie)
  .then(console.log);
