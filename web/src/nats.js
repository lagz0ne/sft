import { connect, StringCodec } from "nats.ws";

let nc;
const sc = StringCodec();

export async function connectNats() {
  const port = location.port || (location.protocol === "https:" ? 443 : 80);
  nc = await connect({ servers: `ws://${location.hostname}:${port}/nats` });
  return nc;
}

export async function requestSpec() {
  const msg = await nc.request("sft.spec", undefined, { timeout: 5000 });
  return JSON.parse(sc.decode(msg.data));
}

export async function requestRender() {
  const msg = await nc.request("sft.render", undefined, { timeout: 5000 });
  return JSON.parse(sc.decode(msg.data));
}

export function subscribeChanges(callback) {
  nc.subscribe("sft.changes", { callback: () => callback() });
}
