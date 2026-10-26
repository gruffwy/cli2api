import test from "node:test";
import assert from "node:assert/strict";
import { buildPlainChatBody } from "../src/plaintext.mjs";

test("Qoder defaults to 128K while preserving explicit output limits", () => {
  for (const [limit, expected] of [[undefined, 128000], [0, 128000], [24, 24], [8192, 8192]]) {
    const body = buildPlainChatBody({
      messages: [{ role: "user", content: "hi" }],
      model: "qwen3.8-max",
      ...(limit === undefined ? {} : { maxTokens: limit }),
    });
    assert.equal(body.parameters.max_tokens, expected);
  }
});
