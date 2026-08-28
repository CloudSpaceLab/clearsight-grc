export type EncryptedRecoveryEnvelope = {
  version: 1;
  algorithm: "AES-GCM";
  iv: ArrayBuffer;
  ciphertext: ArrayBuffer;
  expiresAt: string;
  schemaVersion: number;
};

export interface RecoveryStore {
  get(key: string): Promise<EncryptedRecoveryEnvelope | undefined>;
  put(key: string, value: EncryptedRecoveryEnvelope): Promise<void>;
  delete(key: string): Promise<void>;
  getOrCreateDeviceKey(key: string): Promise<CryptoKey>;
}

const DATABASE_NAME = "clearsight-capture-recovery";
const DATABASE_VERSION = 1;
const ENVELOPE_STORE = "envelopes";
const KEY_STORE = "keys";

export class IndexedDBRecoveryStore implements RecoveryStore {
  constructor(private readonly databaseName = DATABASE_NAME) {}

  async get(key: string): Promise<EncryptedRecoveryEnvelope | undefined> {
    const database = await this.open();
    try {
      return await requestResult<EncryptedRecoveryEnvelope | undefined>(
        database.transaction(ENVELOPE_STORE, "readonly").objectStore(ENVELOPE_STORE).get(key),
      );
    } finally {
      database.close();
    }
  }

  async put(key: string, value: EncryptedRecoveryEnvelope): Promise<void> {
    const database = await this.open();
    try {
      const transaction = database.transaction(ENVELOPE_STORE, "readwrite");
      transaction.objectStore(ENVELOPE_STORE).put(value, key);
      await transactionDone(transaction);
    } finally {
      database.close();
    }
  }

  async delete(key: string): Promise<void> {
    const database = await this.open();
    try {
      const transaction = database.transaction(ENVELOPE_STORE, "readwrite");
      transaction.objectStore(ENVELOPE_STORE).delete(key);
      await transactionDone(transaction);
    } finally {
      database.close();
    }
  }

  async getOrCreateDeviceKey(key: string): Promise<CryptoKey> {
    const database = await this.open();
    try {
      const existing = await requestResult<CryptoKey | undefined>(
        database.transaction(KEY_STORE, "readonly").objectStore(KEY_STORE).get(key),
      );
      if (existing) return existing;

      const candidate = await crypto.subtle.generateKey({ name: "AES-GCM", length: 256 }, false, ["encrypt", "decrypt"]);
      const transaction = database.transaction(KEY_STORE, "readwrite");
      const store = transaction.objectStore(KEY_STORE);
      const current = await requestResult<CryptoKey | undefined>(store.get(key));
      if (current) {
        await transactionDone(transaction);
        return current;
      }
      store.put(candidate, key);
      await transactionDone(transaction);
      return candidate;
    } finally {
      database.close();
    }
  }

  private open(): Promise<IDBDatabase> {
    return new Promise((resolve, reject) => {
      const request = indexedDB.open(this.databaseName, DATABASE_VERSION);
      request.onupgradeneeded = () => {
        const database = request.result;
        if (!database.objectStoreNames.contains(ENVELOPE_STORE)) database.createObjectStore(ENVELOPE_STORE);
        if (!database.objectStoreNames.contains(KEY_STORE)) database.createObjectStore(KEY_STORE);
      };
      request.onsuccess = () => resolve(request.result);
      request.onerror = () => reject(request.error ?? new Error("IndexedDB could not be opened"));
      request.onblocked = () => reject(new Error("IndexedDB upgrade is blocked"));
    });
  }
}

export class MemoryRecoveryStore implements RecoveryStore {
  private readonly envelopes = new Map<string, EncryptedRecoveryEnvelope>();
  private readonly keys = new Map<string, CryptoKey>();

  async get(key: string): Promise<EncryptedRecoveryEnvelope | undefined> {
    return this.envelopes.get(key);
  }

  async put(key: string, value: EncryptedRecoveryEnvelope): Promise<void> {
    this.envelopes.set(key, value);
  }

  async delete(key: string): Promise<void> {
    this.envelopes.delete(key);
  }

  async getOrCreateDeviceKey(key: string): Promise<CryptoKey> {
    const existing = this.keys.get(key);
    if (existing) return existing;
    const created = await crypto.subtle.generateKey({ name: "AES-GCM", length: 256 }, false, ["encrypt", "decrypt"]);
    this.keys.set(key, created);
    return created;
  }

  readRaw(key: string): string | undefined {
    const value = this.envelopes.get(key);
    if (!value) return undefined;
    return JSON.stringify({
      ...value,
      iv: Array.from(new Uint8Array(value.iv)),
      ciphertext: Array.from(new Uint8Array(value.ciphertext)),
    });
  }

  replaceRaw(key: string, value: EncryptedRecoveryEnvelope): void {
    this.envelopes.set(key, value);
  }
}

function requestResult<T>(request: IDBRequest<T>): Promise<T> {
  return new Promise((resolve, reject) => {
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error ?? new Error("IndexedDB request failed"));
  });
}

function transactionDone(transaction: IDBTransaction): Promise<void> {
  return new Promise((resolve, reject) => {
    transaction.oncomplete = () => resolve();
    transaction.onerror = () => reject(transaction.error ?? new Error("IndexedDB transaction failed"));
    transaction.onabort = () => reject(transaction.error ?? new Error("IndexedDB transaction aborted"));
  });
}
