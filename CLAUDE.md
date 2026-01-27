# Core Principles

* we want to make the product that people love
* we will never compromise on performance as a first class citizen
* we will ensure atleast 85% test coverage
* comments should be about WHY. not WHAT


# Little Proofs

1. Monotonicity/Immutability - Prefer one-directional state changes; immutable objects can't be corrupted                                                                  
2. Pre/Post-conditions - Define what must be true before and after functions run                                                                                           
3. Invariants - Identify what must always remain true (like debits = credits in accounting)                                                                                
4. Isolation - Minimize "blast radius" of changes with structural firewalls                                                                                                
5. Induction - For recursion, prove base case + inductive step                                                                                                             
                                                                                                                                                                           
Core insight: Judge code quality by how easily you can reason about its correctness.  

