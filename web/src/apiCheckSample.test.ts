import { describe, expect, it } from "vitest";
import { validateAPICheckSample } from "./apiCheckSample";

describe("API check sample", () => {
  it("accepts false and zero as results", () => {
    expect(validateAPICheckSample('{"result":false,"message":"Not enabled"}', "boolean")).toBeNull();
    expect(validateAPICheckSample('{"result":0}', "number")).toBeNull();
  });

  it("requires the selected result type and a text message when supplied", () => {
    expect(validateAPICheckSample('{"result":"true"}', "boolean")).toMatch(/true or false/);
    expect(validateAPICheckSample('{"result":true}', "number")).toMatch(/number/);
    expect(validateAPICheckSample('{"result":true,"message":null}', "boolean")).toMatch(/message/);
    expect(validateAPICheckSample('{"result":true,"message":1}', "boolean")).toMatch(/message/);
  });

  it("rejects invalid JSON, arrays, missing results and overly long messages", () => {
    expect(validateAPICheckSample('{', "boolean")).toMatch(/JSON/);
    expect(validateAPICheckSample('[]', "boolean")).toMatch(/object/);
    expect(validateAPICheckSample('{}', "boolean")).toMatch(/result/);
    expect(validateAPICheckSample(JSON.stringify({ result: true, message: "a".repeat(1001) }), "boolean")).toMatch(/1,000/);
    expect(validateAPICheckSample('{"result":1e999}', "number")).toMatch(/finite/);
    expect(validateAPICheckSample('{"result":true,"other":1}', "boolean")).toMatch(/unsupported/);
    expect(validateAPICheckSample('{"result":true,"message":"\\u0000"}', "boolean")).toMatch(/message/);
  });

  it("rejects duplicate fields including escaped names", () => {
    expect(validateAPICheckSample('{"result":false,"result":true}', "boolean")).toMatch(/duplicate/i);
    expect(validateAPICheckSample('{"result":false,"res\\u0075lt":true}', "boolean")).toMatch(/duplicate/i);
    expect(validateAPICheckSample('{"result":true,"message":"a","message":"b"}', "boolean")).toMatch(/duplicate/i);
    expect(validateAPICheckSample('{"result":true,"message":"result: \\"quoted\\""}', "boolean")).toBeNull();
  });

  it("rejects examples over the backend size limit", () => {
    const oversized = JSON.stringify({ result: true, message: "é".repeat(33000) });
    expect(validateAPICheckSample(oversized, "boolean")).toMatch(/too large/i);
  });
});
