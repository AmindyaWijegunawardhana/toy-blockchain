## 1.2 Difficulty versus Effort Benchmarks

[cite_start]The mining engine was calibrated across progressive cryptographic difficulty targets[cite: 61, 93]. [cite_start]Each target increments the required count of matching leading zero hexadecimal digits ($N$) required for the Proof-of-Work threshold[cite: 61, 123]. [cite_start]The performance profile collected on a local architecture is detailed below[cite: 9, 93]:

| Difficulty Target ($N$) | Nonce Found | Time Elapsed |
| :--- | :--- | :--- |
| **1** | 12 | 0s |
| **2** | 470 | 858.8 µs |
| **3** | 2,557 | 2.58 ms |
| **4** | 24,577 | 29.54 ms |
| **5** | 587,747 | 649.27 ms |

### Mathematical Trend Line
[cite_start]The relationship between difficulty and effort is exponential ($O(16^N)$), not linear[cite: 94]. Because each character in a hex string represents 4 bits, adding an additional zero digit restricts the target space by a factor of 16 ($16^1, 16^2, 16^3...$). [cite_start]The computer is forced to test a dramatically growing space of arbitrary nonces to land a valid solution, making the mining difficulty predictable and highly scalable[cite: 61, 93].

---

## 1.3 Architecture Design Write-Up

### Hashing Configuration
[cite_start]We utilize standard SHA-256 mapping via Go’s standard library (`crypto/sha256`)[cite: 52, 105]. [cite_start]To guarantee strict determinism, fields are passed out using an isolated `HashInput` abstraction tracking fields sequentially (`Index` -> `Timestamp` -> `Transactions` -> `PrevHash` -> `Nonce`)[cite: 46, 52]. [cite_start]The structural `Hash` field itself is intentionally omitted from this serialization loop to prevent mathematical circular dependencies[cite: 52].

### Chain-Wide Integrity Guarantees
[cite_start]The integrity of the ledger depends on back-linked dependencies[cite: 25, 66]. [cite_start]Because Block $X$ embeds the exact string of `previous.Hash` inside its structure, changing the historical transactions of Block $X-1$ rewrites its own hash[cite: 46, 66]. [cite_start]This instantly causes a structural mismatch in Block $X$’s `PrevHash` property[cite: 66]. [cite_start]Consequently, to forge a single entry undetected, an attacker must re-calculate every single Proof-of-Work nonce for all succeeding blocks across the system[cite: 61, 66, 97].

---

## 2. Discussion Questions

### 2.1 The Impracticality of Tampering in Production
[cite_start]In our local toy simulator, tampering is straightforward because the state lives entirely within a localized single-process memory heap or individual JSON file[cite: 9, 30, 74]. [cite_start]In a production blockchain network, tampering becomes practically impossible due to distributed consensus and the longest-chain rule[cite: 38, 97].

[cite_start]Even if an attacker possesses the raw processing power to re-mine subsequent blocks locally on their own node, their isolated modified ledger will be instantly rejected by the rest of the network[cite: 38, 97]. [cite_start]Peers enforce peer-to-peer verification protocols; they will only accept a chain that matches the majority consensus and represents the maximum cumulative Proof-of-Work difficulty[cite: 38, 66, 97]. [cite_start]Overriding this requires a 51% attack, where a malicious entity must control more computing hardware than the remainder of the global network combined—rendering tampering financially and structurally prohibitive[cite: 97].

### 2.2 Consensus Alternatives: Proof-of-Stake (PoS)

* [cite_start]**Mechanism:** Instead of node operators racing to solve resource-intensive arithmetic loops via processing hardware, blocks are produced by selected validators who commit a quantity of native token capital ("stake") as a collateral deposit[cite: 99, 100].
* [cite_start]**Advantage over PoW:** Massive operational sustainability[cite: 100]. [cite_start]It completely eliminates the massive energy consumption, carbon footprints, and specialized hardware demands (ASICs) that characterize conventional Proof-of-Work setups[cite: 100].
* [cite_start]**Drawback over PoW:** Accumulation centralization risk ("the rich get richer")[cite: 100]. [cite_start]Because selection probability scales proportionally with the quantity of tokens locked up, wealthy participants accumulate higher validator control and collect the majority of network rewards, threatening decentralized governance[cite: 100].

### 2.3 Structural Gaps: Toy vs. Production

* [cite_start]**Network Layer Consensus:** The toy is localized to a single process[cite: 9, 30]. [cite_start]Production systems use distributed peer-to-peer gossip communication frameworks (like Libp2p) to keep thousands of isolated ledger nodes in sync[cite: 38, 101].
* [cite_start]**Asymmetric Transaction Security:** Our model uses unverified text tags for accounts[cite: 56]. [cite_start]Real-world chains mandate public/private key-pair typography (like ECDSA or Ed25519) so every transfer carries a cryptographic digital signature proving structural authorization[cite: 39, 101].
* [cite_start]**Optimized Structural Hashing (Merkle Trees):** We serialize raw transaction lists directly into block data[cite: 46, 52]. [cite_start]Production environments build hierarchical Merkle Trees, maintaining a single Merkle Root inside the block header to facilitate rapid, light-weight asset validation without parsing full transaction arrays[cite: 101, 154].

#### Architectural Sketch: Implementing a Merkle Root
[cite_start]To swap out raw transaction listing hashes for an elegant Merkle Root design, the core block processing loop would be updated as follows[cite: 102, 154]:

```text
               [ Merkle Root ]  -> Stored directly in Block Header
                /            \
         [ Hash AB ]      [ Hash CD ]
          /       \        /       \
       [Tx A]   [Tx B]  [Tx C]   [Tx D] -> Raw Transaction Hashing