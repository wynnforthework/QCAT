"use client";

import { useState } from "react";
import api from "@/lib/api";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";

export default function SelectorPage() {
  const [symbol, setSymbol] = useState("BTCUSDT");
  const [decision, setDecision] = useState<any>(null);
  const [stats, setStats] = useState<any>(null);
  const [shadowEnabled, setShadowEnabled] = useState(true);
  const [whitelist, setWhitelist] = useState("BTCUSDT,ETHUSDT");

  const [sample, setSample] = useState({
    strategy_id: "",
    pnl: 0,
    return: 0,
    drawdown: 0,
    win: true,
    cost: 0,
    duration_ms: 60000,
  } as any);

  const fetchDecision = async () => {
    try { setDecision(await api.getSelectorDecision(symbol)); } catch (e) { setDecision({ error: String(e) }); }
  };
  const fetchStats = async () => {
    try { setStats(await api.getSelectorStats(symbol)); } catch (e) { setStats({ error: String(e) }); }
  };
  const applyShadow = async () => {
    const symbols = whitelist.split(",").map(s => s.trim()).filter(Boolean);
    await api.setSelectorShadow(shadowEnabled, symbols);
  };
  const pushSample = async () => {
    await api.postSelectorSample({ symbol, ...sample });
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Selector 管理</h1>

      <Card className="p-4 space-y-4">
        <h2 className="text-lg font-semibold">查询决策 / 统计</h2>
        <div className="flex items-center gap-2">
          <Label className="w-16">Symbol</Label>
          <Input value={symbol} onChange={(e) => setSymbol(e.target.value)} className="w-48" />
          <Button onClick={fetchDecision}>获取决策</Button>
          <Button variant="secondary" onClick={fetchStats}>获取统计</Button>
        </div>
        <Separator />
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <pre className="bg-gray-50 p-3 rounded border overflow-x-auto"><code>{JSON.stringify(decision, null, 2)}</code></pre>
          <pre className="bg-gray-50 p-3 rounded border overflow-x-auto"><code>{JSON.stringify(stats, null, 2)}</code></pre>
        </div>
      </Card>

      <Card className="p-4 space-y-4">
        <h2 className="text-lg font-semibold">影子模式设置</h2>
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <Switch checked={shadowEnabled} onCheckedChange={setShadowEnabled} />
            <Label>启用影子模式</Label>
          </div>
          <div className="flex items-center gap-2">
            <Label className="whitespace-nowrap">白名单(逗号分隔)</Label>
            <Input value={whitelist} onChange={(e) => setWhitelist(e.target.value)} className="w-96" />
          </div>
          <Button onClick={applyShadow}>应用</Button>
        </div>
      </Card>

      <Card className="p-4 space-y-4">
        <h2 className="text-lg font-semibold">写入样本（调试）</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          <div>
            <Label>Strategy ID</Label>
            <Input value={sample.strategy_id} onChange={(e) => setSample({ ...sample, strategy_id: e.target.value })} />
          </div>
          <div>
            <Label>PNL</Label>
            <Input type="number" value={sample.pnl} onChange={(e) => setSample({ ...sample, pnl: Number(e.target.value) })} />
          </div>
          <div>
            <Label>Return</Label>
            <Input type="number" value={sample.return} onChange={(e) => setSample({ ...sample, return: Number(e.target.value) })} />
          </div>
          <div>
            <Label>Drawdown</Label>
            <Input type="number" value={sample.drawdown} onChange={(e) => setSample({ ...sample, drawdown: Number(e.target.value) })} />
          </div>
          <div>
            <Label>Win</Label>
            <select className="border rounded px-2 py-1 w-full" value={sample.win ? "true" : "false"} onChange={(e) => setSample({ ...sample, win: e.target.value === "true" })}>
              <option value="true">true</option>
              <option value="false">false</option>
            </select>
          </div>
          <div>
            <Label>Cost</Label>
            <Input type="number" value={sample.cost} onChange={(e) => setSample({ ...sample, cost: Number(e.target.value) })} />
          </div>
          <div>
            <Label>Duration(ms)</Label>
            <Input type="number" value={sample.duration_ms} onChange={(e) => setSample({ ...sample, duration_ms: Number(e.target.value) })} />
          </div>
        </div>
        <div>
          <Button onClick={pushSample}>提交样本</Button>
        </div>
      </Card>
    </div>
  );
}


