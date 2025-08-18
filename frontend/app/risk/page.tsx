"use client";

import { RiskDashboard } from "@/components/risk/risk-dashboard";

export default function RiskPage() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">风险控制</h1>
          <p className="text-gray-600 mt-1">风险监控、限额管理和预警系统</p>
        </div>
      </div>
      
      <RiskDashboard />
    </div>
  );
}