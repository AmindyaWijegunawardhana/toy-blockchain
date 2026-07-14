# Research & Evaluation Report: Toy Blockchain & Ledger Simulator

Role: Software Engineering Intern (Backend, Go)

Date: July 2026

Author: Amindya Nimeshani

## 1. Required Investigations

### 1.1 Tamper-Evidence & Integrity Constraints

To evaluate the cryptographic defense capabilities of the architectural ledger design, an adversarial attack simulation was performed. A transaction value within a confirmed historical block (Block 1) was manually altered from its original honest value to a fraudulent amount.

Upon running the full-chain `ValidateChain()` execution loop following the data modification, the engine instantly identified the security breach and flagged the ledger state as invalid.

**Detection Mechanism:** The compromise was detected by the data hash recomputation check.

**Technical Analysis:** A block's hash is generated over a stable, deterministic serialization of its core internal state properties: `Index`, `Timestamp`, `Transactions`, `PrevHash`, and `Nonce`. Due to the avalanche effect inherent to the SHA-256 algorithm, modifying even a single character or digit inside a transaction completely mutates the resulting hexadecimal digest. Because the block's recorded hash no longer matches the recomputed state hash, the validation engine detects the anomaly immediately and returns the exact index of the compromised block.

**Empirical Simulation Log:**

```text
ATTACK SCENARIO: Modifying historical transaction inside Block 1...
>>> Post-Tamper Status: Valid = false
>>> Identified Offending Block Index: 1
>>> Validator Reason: hash mismatch at index 1: block data was modified
```

### 1.2 Difficulty versus Effort Benchmarks

The mining engine was calibrated across progressive cryptographic difficulty targets. Each target increments the required count of matching leading zero hexadecimal digits ($N$) required for the Proof-of-Work threshold. The performance profile collected on a local architecture is detailed below.

| Difficulty Target ($N$) | Nonce Found | Time Elapsed |
| :--- | :--- | :--- |
| **1** | 12 | 0s |
| **2** | 470 | 858.8 microseconds |
| **3** | 2,557 | 2.58 ms |
| **4** | 24,577 | 29.54 ms |
| **5** | 587,747 | 649.27 ms |

#### Mathematical Trend Line

The relationship between difficulty and effort is exponential ($O(16^N)$), not linear. Because each character in a hex string represents 4 bits, adding an additional zero digit restricts the target space by a factor of 16 ($16^1, 16^2, 16^3...$). The computer is forced to test a dramatically growing space of arbitrary nonces to land a valid solution, making the mining difficulty predictable and highly scalable.

### 1.3 Architecture Design Write-Up

#### Hashing Configuration

We utilize standard SHA-256 mapping via Go's standard library (`crypto/sha256`). To guarantee strict determinism, fields are passed out using an isolated `HashInput` abstraction tracking fields sequentially:

`Index` -> `Timestamp` -> `Transactions` -> `PrevHash` -> `Nonce`

The structural `Hash` field itself is intentionally omitted from this serialization loop to prevent mathematical circular dependencies.

#### Chain-Wide Integrity Guarantees

The integrity of the ledger depends on back-linked dependencies. Because Block $X$ embeds the exact string of `previous.Hash` inside its structure, changing the historical transactions of Block $X-1$ rewrites its own hash. This instantly causes a structural mismatch in Block $X$'s `PrevHash` property. Consequently, to forge a single entry undetected, an attacker must re-calculate every single Proof-of-Work nonce for all succeeding blocks across the system.

## 2. Discussion Questions

### 2.1 The Impracticality of Tampering in Production

In our local toy simulator, tampering is straightforward because the state lives entirely within a localized single-process memory heap or individual JSON file. In a production blockchain network, tampering becomes practically impossible due to distributed consensus and the longest-chain rule.

Even if an attacker possesses the raw processing power to re-mine subsequent blocks locally on their own node, their isolated modified ledger will be instantly rejected by the rest of the network. Peers enforce peer-to-peer verification protocols; they will only accept a chain that matches the majority consensus and represents the maximum cumulative Proof-of-Work difficulty. Overriding this requires a 51% attack, where a malicious entity must control more computing hardware than the remainder of the global network combined, rendering tampering financially and structurally prohibitive.

### 2.2 Consensus Alternatives: Proof-of-Stake (PoS)

* **Mechanism:** Instead of node operators racing to solve resource-intensive arithmetic loops via processing hardware, blocks are produced by selected validators who commit a quantity of native token capital ("stake") as a collateral deposit.
* **Advantage over PoW:** Massive operational sustainability. It completely eliminates the massive energy consumption, carbon footprints, and specialized hardware demands (ASICs) that characterize conventional Proof-of-Work setups.
* **Drawback over PoW:** Accumulation centralization risk ("the rich get richer"). Because selection probability scales proportionally with the quantity of tokens locked up, wealthy participants accumulate higher validator control and collect the majority of network rewards, threatening decentralized governance.

### 2.3 Structural Gaps: Toy vs. Production

* **Network Layer Consensus:** The toy is localized to a single process. Production systems use distributed peer-to-peer gossip communication frameworks (like Libp2p) to keep thousands of isolated ledger nodes in sync.
* **Asymmetric Transaction Security:** Our model uses unverified text tags for accounts. Real-world chains mandate public/private key-pair cryptography (like ECDSA or Ed25519) so every transfer carries a cryptographic digital signature proving structural authorization.
* **Optimized Structural Hashing (Merkle Trees):** We serialize raw transaction lists directly into block data. Production environments build hierarchical Merkle Trees, maintaining a single Merkle Root inside the block header to facilitate rapid, lightweight asset validation without parsing full transaction arrays.

#### Architectural Sketch: Implementing a Merkle Root

To swap out raw transaction listing hashes for an elegant Merkle Root design, the core block processing loop would be updated as follows:

```text
               [ Merkle Root ]  -> Stored directly in Block Header
                /            \
         [ Hash AB ]      [ Hash CD ]
          /       \        /       \
       [Tx A]   [Tx B]  [Tx C]   [Tx D] -> Raw Transaction Hashing
```

Each individual transaction inside a block is hashed once using SHA-256. The system pairs adjacent leaves sequentially and hashes their string components together (`Hash(Hash_A + Hash_B)`). This cryptographic pairing continues up the tree until a single master hash remains, the Merkle Root. The structural `HashInput` layout would then hold a single, uniform 32-byte `MerkleRoot` string instead of the full transaction slice, allowing lightweight clients to confirm inclusion without loading complete blocks.
Research & Evaluation Report: Toy Blockchain & Ledger SimulatorRole: Software Engineering Intern (Backend, Go)Date: July 2026Author: Amindya Nimeshani1. Required Investigations1.1 Tamper-Evidence & Integrity ConstraintsTo evaluate the cryptographic defense capabilities of the architectural ledger design, an adversarial attack simulation was performed. A transaction value within a confirmed historical block (Block 1) was manually altered from its original honest value to a fraudulent amount.Upon running the full-chain ValidateChain() execution loop following the data modification, the engine instantly identified the security breach and flagged the ledger state as invalid:Detection Mechanism: The compromise was detected by the Data Hash Recomputation Check.Technical Analysis: A block's hash is generated over a stable, deterministic serialization of its core internal state properties: Index, Timestamp, Transactions, PrevHash, and Nonce. Due to the avalanche effect inherent to the SHA-256 algorithm, modifying even a single character or digit inside a transaction completely mutates the resulting hexadecimal digest. Because the block's recorded hash no longer matches the recomputed state hash, the validation engine detects the anomaly immediately and returns the exact index of the compromised block.Empirical Simulation Log:Plaintext[5] ATTACK SCENARIO: Modifying historical transaction inside Block 1...
 >>> Post-Tamper Status: Valid = false
 >>> Identified Offending Block Index: 1
 >>> Validator Reason: hash mismatch at index 1: block data was modified
1.2 Difficulty versus Effort BenchmarksThe mining engine was calibrated across progressive cryptographic difficulty targets. Each progressive difficulty level increments the required count of matching leading zero hexadecimal digits ($N$) mandated for the Proof-of-Work threshold. The empirical performance profile collected on the local hardware architecture is detailed below:Difficulty Target (N)Nonce FoundTime Elapsed1120s2470858.8 µs32,5572.58 ms424,57729.54 ms5587,747649.27 msMathematical Trend AnalysisThe relationship between difficulty scaling and computational effort is exponential ($O(16^N)$), rather than linear. Because each character in a hexadecimal string represents a 4-bit nibble, adding an additional required zero digit restricts the valid target hash landscape by a factor of 16 ($16^1, 16^2, 16^3...$). The system is forced to calculate a exponentially expanding space of arbitrary nonces to land a valid solution, making the mining difficulty predictable and highly scalable under load.1.3 Architecture Design Write-UpHashing ConfigurationWe utilize standard SHA-256 hashing via Go’s standard library (crypto/sha256). To guarantee strict mathematical determinism across executions, block fields are extracted into an isolated HashInput abstraction layer that orders fields sequentially:$$\text{Index} \longrightarrow \text{Timestamp} \longrightarrow \text{Transactions} \longrightarrow \text{PrevHash} \longrightarrow \text{Nonce}$$The structural Hash field itself is intentionally omitted from this serialization loop to prevent mathematical circular dependencies.Chain-Wide Integrity GuaranteesThe integrity of the ledger depends on sequential back-linked dependencies. Because Block $X$ explicitly embeds the exact string of previous.Hash inside its structure, changing the historical transactions of Block $X-1$ rewrites its own hash. This instantly causes a structural mismatch in Block $X$’s PrevHash property. Consequently, to forge a single entry undetected, an attacker must re-calculate every single Proof-of-Work nonce for all succeeding blocks across the system.2. Discussion Questions2.1 The Impracticality of Tampering in ProductionIn our local single-process simulator, tampering is straightforward because the state lives entirely within a localized memory heap or individual JSON file. In a production decentralized blockchain network, tampering becomes practically impossible due to distributed consensus protocols and the longest-chain rule.Even if an attacker possesses the raw processing power to re-mine subsequent blocks locally on their own isolated node, their modified ledger will be instantly rejected by the network peers. Network nodes enforce strict peer-to-peer verification protocols; they will only accept a chain that matches the majority consensus and represents the maximum cumulative Proof-of-Work difficulty. Overriding this requires a 51% attack, where a malicious entity must control more computing hardware than the remainder of the global network combined—rendering tampering financially and structurally prohibitive.2.2 Consensus Alternatives: Proof-of-Stake (PoS)Mechanism: Instead of node operators racing to solve resource-intensive arithmetic loops via processing hardware, blocks are produced by selected validators who commit a quantity of native token capital ("stake") as a collateral deposit.Advantage over PoW: Massive operational sustainability. It completely eliminates the massive energy consumption, carbon footprints, and specialized hardware demands (ASICs) that characterize conventional Proof-of-Work setups.Drawback over PoW: Accumulation centralization risk ("the rich get richer"). Because selection probability scales proportionally with the quantity of tokens locked up, wealthy participants accumulate higher validator control and collect the majority of network rewards, threatening decentralized governance.2.3 Structural Gaps: Toy vs. ProductionNetwork Layer Consensus: The toy is localized to a single process. Production systems use distributed peer-to-peer gossip communication frameworks (like Libp2p) to keep thousands of isolated ledger nodes in sync.Asymmetric Transaction Security: Our model uses unverified text tags for accounts. Real-world chains mandate public/private key-pair typography (like ECDSA or Ed25519) so every transfer carries a cryptographic digital signature proving structural authorization.Optimized Structural Hashing (Merkle Trees): We serialize raw transaction lists directly into block data. Production environments build hierarchical Merkle Trees, maintaining a single Merkle Root inside the block header to facilitate rapid, light-weight asset validation without parsing full transaction arrays.Architectural Sketch: Implementing a Merkle RootTo swap out raw transaction listing hashes for an elegant Merkle Root design, the core block processing loop would be updated as follows:Plaintext               [ Merkle Root ]  -> Stored directly in Block Header
                /            \
         [ Hash AB ]      [ Hash CD ]
          /       \        /       \
       [Tx A]   [Tx B]  [Tx C]   [Tx D] -> Raw Transaction Hashing
Each individual transaction inside a block is hashed once using SHA-256.The system pairs adjacent leaves up sequentially and hashes their string components together ($Hash(Hash_A + Hash_B)$).This cryptographic pairing continues up the tree until a single master hash remains—the Merkle Root.The structural HashInput layout will no longer include the variable-length transaction slice; instead, it will hold a single, uniform 32-byte MerkleRoot string. This allows lightweight clients to safely confirm a transaction's existence using a minimal path proof without loading full blocks.