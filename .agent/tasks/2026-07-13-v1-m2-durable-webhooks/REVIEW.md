# Review

Approved after integration review.

The processor now propagates cancellation, contains router panics, preserves fixed worker capacity, and prunes completed payloads after 30 days in bounded batches. Signature failure and oversized payloads never reach storage; successful HTTP acceptance follows the unique durable insert. Focused, full, and race tests pass.
