"use client";
import React, { useEffect, useState } from "react";
import { Card } from "../components/ui/card";
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, PieChart, Pie, Cell, Legend } from "recharts";
import { ToggleGroup, ToggleGroupItem } from "../components/ui/toggle-group";

const COLORS = ["#0088FE", "#00C49F", "#FFBB28", "#FF8042", "#E44D26", "#AF19FF", "#A14EBF", "#FFA500"];

export default function Dashboard() {
  const [nodes, setNodes] = useState([]);
  const [metrics, setMetrics] = useState({ node: {}, process: {}, functions: {} });
  const [chartType, setChartType] = useState("bar");

  useEffect(() => {
    fetch("http://localhost:8080/cluster-info")
      .then(res => res.json())
      .then(data => setNodes(Object.entries(data)));

    fetch("http://localhost:8080/metrics")
      .then(res => res.json())
      .then(data => {
        console.log("Dati ricevuti dal backend:", data);
        if (Array.isArray(data)) {
          const categorizedMetrics = { node: { cpu: [], mem: [] }, process: { cpu: [], mem: [] }, functions: { cpu: [], mem: [] } }; // TODO: todo, le metriche sedge vanno gestite
          data.forEach(({ data: metricData }) => {
            if (Array.isArray(metricData?.result)) {
              metricData.result.forEach(({ metric, value }) => {
                const metricValue = parseFloat(value[1]) || 0;
                if (metric.__name__.includes("node")) {
                  if (metric.__name__.includes("cpu")) categorizedMetrics.node.cpu.push({ name: `${metric.__name__}`, value: metricValue });
                  else if (metric.__name__.includes("mem")) categorizedMetrics.node.mem.push({ name: `${metric.__name__}`, value: metricValue });
                } else if (metric.__name__.includes("namedprocess")) {
                  if (metric.__name__.includes("cpu")) categorizedMetrics.process.cpu.push({ name: `${metric.__name__}:${metric.mode}`, value: metricValue, mode: metric.mode });
                  else if (metric.__name__.includes("mem")) categorizedMetrics.process.mem.push({ name: `${metric.__name__}:${metric.mode}` , value: metricValue, mode: metric.mode });
                } else if (metric.__name__.includes("sedge")) {
                  if (metric.__name__.includes("cpu")) categorizedMetrics.functions.cpu.push({ name: `${metric.__name__}:${metric.function}`, value: metricValue });
                  else if (metric.__name__.includes("mem")) categorizedMetrics.functions.mem.push({ name: `${metric.__name__}:${metric.function}`, value: metricValue });
                }
              });
            }
          });
          setMetrics(categorizedMetrics);
        }
      });
  }, []);

  return (
    <div className="p-5">
      <h1 className="text-2xl font-bold mb-5">Serverledge Dashboard</h1>
      
      <Card className="mb-5 p-4">
        <h2 className="text-xl font-semibold" style={{color:"black"}}>Clusters Info</h2>
        <ul>
          {nodes.map(([key, value]) => (
            <li key={key} className="border-b py-2">{key}: {value}</li>
          ))}
        </ul>
      </Card>
      
      <ToggleGroup className="mb-4" value={chartType} onValueChange={setChartType}>
        <ToggleGroupItem value="bar">Bar Chart</ToggleGroupItem>
        <ToggleGroupItem value="pie">Pie Chart</ToggleGroupItem>
      </ToggleGroup>
      
      {Object.entries(metrics).map(([category, subMetrics]) => (
        <div key={category}>
          <h2 className="text-2xl font-bold mt-5" style={{color:"white"}}>{category.toUpperCase()}</h2>
          {
            Object.entries(subMetrics).map(([type, data], index) => (
              <Card className="p-4 mb-5" key={index}>
                <h3 className="text-xl font-semibold" style={{color:"black"}}>{type.toUpperCase()}</h3>
                <ResponsiveContainer width="100%" height={300}>
                  {chartType === "bar" ? (
                    <BarChart data={data}>
                      <XAxis dataKey="name" />
                      <YAxis />
                      <Tooltip />
                      <Bar dataKey="value" fill={COLORS[index % COLORS.length]} />
                    </BarChart>
                  ) : (
                    <PieChart>
                      <Pie data={data} dataKey="value" nameKey="name" cx="50%" cy="50%" outerRadius={100} innerRadius={50} label>
                        {data.map((_, i) => (
                          <Cell key={`cell-${i}`} fill={COLORS[i % COLORS.length]} />
                        ))}
                      </Pie>
                      <Tooltip />
                      <Legend />
                    </PieChart>
                  )}
                </ResponsiveContainer>
              </Card>
            ))
         }
        </div>
      ))}
    </div>
  );
}
