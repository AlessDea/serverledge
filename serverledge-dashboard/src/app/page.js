"use client";

import React, { useEffect, useState } from "react";
import { Card, CardContent } from "../components/ui/card";
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, PieChart, Pie, Cell, Legend } from "recharts";
import { ToggleGroup, ToggleGroupItem } from "../components/ui/toggle-group";

const COLORS = ["#0088FE", "#00C49F", "#FFBB28", "#FF8042", "#E44D26", "#AF19FF"];

export default function Dashboard() {
  const [nodes, setNodes] = useState([]);
  const [metrics, setMetrics] = useState([]);
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
          const metrics = data.flatMap(({ data: metricData }) => {
            // Verifica che ci sia il campo 'result' e che sia un array
            if (Array.isArray(metricData?.result)) {
              return metricData.result.map(({ metric, value }) => ({
                name: metric.__name__,
                instance: metric.instance,
                job: metric.job,
                timestamp: value[0], // Primo valore (timestamp)
                metricValue: parseFloat(value[1]),  // Secondo valore (valore della metrica)
              }));
            }
            return [];
          });
          setMetrics(metrics);          
        }
      })
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
      
      {/* Metriche
      <Card className="p-4">
        <h2 className="text-xl font-semibold" style={{color:"black"}}>Metrics</h2>
        <ResponsiveContainer width="80%" height={300}>
          <BarChart data={metrics.map(m => ({
            instance: m.name,  // Usa `m.instance` qui
            value: parseFloat(m.metricValue)/10000  // Usa `m.metricValue` qui
          }))}>
            <XAxis dataKey="instance" />
            <YAxis />
            <Tooltip />
            <Bar dataKey="value" fill="#4F46E5" />
          </BarChart>
        </ResponsiveContainer>
      </Card>
          <br></br> */}
      <Card className="p-4">
        <h2 className="text-xl font-semibold" style={{color:"black"}}>Metrics</h2>
        <ToggleGroup className="mb-4" value={chartType} onValueChange={setChartType}>
          <ToggleGroupItem value="bar">Bar Chart</ToggleGroupItem>
          <ToggleGroupItem value="pie">Pie Chart</ToggleGroupItem>
        </ToggleGroup>
        
        <ResponsiveContainer width="100%" height={300}>
          {chartType === "bar" ? (
            <BarChart data={metrics.map(m => ({ 
                        name: m.name, 
                        value: parseFloat(m.metricValue) || 0
                      }))}>
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip />
              <Bar dataKey="value" fill="#4F46E5" />
              {metrics.map((_, index) => (
                  <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                ))}
            </BarChart>
          ) : (
            <PieChart>
              <Pie
                data={metrics.map((m, index) => ({ 
                  instance: m.name, 
                  value: parseFloat(m.metricValue) || 0
                }))}
                dataKey="value"
                nameKey="instance"
                cx="50%"
                cy="50%"
                outerRadius={100}
                innerRadius={50}
                fill="#8884d8"
                label
              >
                {metrics.map((_, index) => (
                  <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                ))}
              </Pie>
              <Tooltip />
              <Legend />
            </PieChart>
          )}
        </ResponsiveContainer>
      </Card>
    </div>
  );
}


// "use client";

// import React, { useEffect, useState } from "react";
// import { Card, CardContent } from "../components/ui/card";
// import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, PieChart, Pie, Cell, Legend } from "recharts";

// export default function Dashboard() {
//   const [nodes, setNodes] = useState([]);
//   const [metrics, setMetrics] = useState([]);

//   useEffect(() => {
//     // Retrieve cluster info
//     fetch("http://localhost:8080/cluster-info")
//       .then(res => {
//         if (!res.ok) throw new Error("Error API Cluster");
//         return res.json();
//       })
//       .then(data => setNodes(Object.entries(data)))
//       .catch(error => console.error("Errore cluster:", error));
  
//     // Retrieve memory nodes metrics
//     fetch("http://localhost:8080/metrics")
//       .then(res => {
//         if (!res.ok) throw new Error("Errore API Metrics");
//         return res.json();
//       })
//       .then(data => {
//         if (data?.data?.result) {
//           const metrics = data.data.result.map(({ metric, value }) => ({
//             name: metric.__name__,
//             instance: metric.instance,
//             job: metric.job,
//             timestamp: value[0],
//             metricValue: parseFloat(value[1]),
//           })).filter(Boolean);
//           setMetrics(metrics);
//         }
//       })
//       // .then(data => {
//       //   console.log("received metrics:", data);
  
//       //   if (data && data.data && data.data.result) {
//       //     const metrics = data.data.result.map(metricItem => {
//       //       console.log("Evaluating metrics:", metricItem);
//       //       const { metric, value } = metricItem;
  
//       //       if (!value || value.length < 2) {
//       //         console.error("Not valid metric's value:", metricItem);
//       //         return null;
//       //       }
  
//       //       const timestamp = value[0];  // timestamp
//       //       const metricValue = value[1]; // valore della metrica (es. "1")
  
//       //       // Cambia la struttura per usare correttamente `instance` direttamente
//       //       return {
//       //         name: metric.__name__,
//       //         instance: metricItem.metric.instance,  // Modifica qui
//       //         job: metric.job,
//       //         timestamp,
//       //         metricValue
//       //       };
//       //     }).filter(Boolean);  // Filtra eventuali valori nulli
  
//       //     console.log("Formatted metrics:", metrics);  // Log delle metriche formattate
//       //     setMetrics(metrics);  // imposta i dati delle metriche formattati
//       //   } else {
//       //     console.error("not valid data:", data);
//       //   }
//       // })
//       .catch(error => console.error("Errore metriche:", error));
//   }, []);
  
//   // Colori personalizzati per le label
//   const COLORS = ["#0088FE", "#00C49F", "#FFBB28", "#FF8042", "#E44D26", "#AF19FF"];

//   return (
//     <div className="p-5">
//       <h1 className="text-2xl font-bold mb-5">Serverledge Dashboard</h1>

//       {/* Info Cluster */}
//       <Card className="mb-5 p-4">
//         <h2 className="text-xl font-semibold" style={{color:"black"}}>Cluster Info</h2>
//         <ul>
//           {nodes.map(([key, value]) => (
//             <li key={key} className="border-b py-2">{key}: {value}</li>
//           ))}
//         </ul>
//       </Card>

//       {/* Metriche */}
//       <Card className="p-4">
//         <h2 className="text-xl font-semibold" style={{color:"black"}}>Metrics</h2>
//         <ResponsiveContainer width="80%" height={300}>
//           <BarChart data={metrics.map(m => ({
//             instance: m.name,  // Usa `m.instance` qui
//             value: parseFloat(m.metricValue)/10000  // Usa `m.metricValue` qui
//           }))}>
//             <XAxis dataKey="instance" />
//             <YAxis />
//             <Tooltip />
//             <Bar dataKey="value" fill="#4F46E5" />
//           </BarChart>
//         </ResponsiveContainer>
//       </Card>

//       {/* Grafico a Ciambella (Doughnut Chart) */}
//       <Card className="p-6">
//         <h2 className="text-xl font-semibold mb-4" style={{color:"black"}}>Distributions</h2>
//         <ResponsiveContainer width="100%" height={300}>
//           <PieChart>
//             <Pie
//                data={metrics.map(m => ({
//                 name: m.name,  // Mostra l'istanza nel tooltip
//                 value: parseFloat(m.metricValue)  // Valore numerico della metrica
//               }))}
//               dataKey="metricValue"
//               nameKey="name"
//               cx="50%"
//               cy="50%"
//               outerRadius={100}
//               innerRadius={50} // Centro vuoto
//               fill="#8884d8"
//               label
//             >
//               {metrics.map((_, index) => (
//                 <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
//               ))}
//             </Pie>
//             <Tooltip />
//             <Legend />
//           </PieChart>
//         </ResponsiveContainer>
//       </Card>

//       {/* Grafico a Torta */}
//       <Card className="p-4 mb-5">
//         <h2 className="text-xl font-semibold" style={{color:"black"}}>Distributions</h2>
//         <ResponsiveContainer width="100%" height={300}>
//           <PieChart>
//             <Pie
//                data={metrics.map(m => ({
//                 name: m.name,  // Mostra l'istanza nel tooltip
//                 value: parseFloat(m.metricValue)  // Valore numerico della metrica
//               }))}
//               dataKey="metricValue"
//               nameKey="instance"
//               cx="50%"
//               cy="50%"
//               outerRadius={100}
//               fill="#8884d8"
//               label
//             >
//               {metrics.map((_, index) => (
//                 <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
//               ))}
//             </Pie>
//             <Tooltip />
//             <Legend />
//           </PieChart>
//         </ResponsiveContainer>
//       </Card>

//     </div>
//   );
// }
