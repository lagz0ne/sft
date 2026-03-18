import { connect, type NatsConnection, StringCodec } from "nats.ws";
import type { Spec, RenderSpec } from "./types";

let nc: NatsConnection | null = null;
const sc = StringCodec();

export async function connectNats(): Promise<NatsConnection> {
	const port = location.port || (location.protocol === "https:" ? "443" : "80");
	nc = await connect({ servers: `ws://${location.hostname}:${port}/nats` });
	return nc;
}

export async function requestSpec(): Promise<Spec> {
	if (!nc) throw new Error("NATS not connected");
	const msg = await nc.request("sft.spec", undefined, { timeout: 5000 });
	return JSON.parse(sc.decode(msg.data));
}

export async function requestRender(): Promise<RenderSpec> {
	if (!nc) throw new Error("NATS not connected");
	const msg = await nc.request("sft.render", undefined, { timeout: 5000 });
	return JSON.parse(sc.decode(msg.data));
}

export function subscribeChanges(callback: () => void): void {
	if (!nc) throw new Error("NATS not connected");
	nc.subscribe("sft.changes", { callback: () => callback() });
}
