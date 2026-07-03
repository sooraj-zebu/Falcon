let cachedEdges=[];

async function getJSON(url){
    const res=await fetch(url);
    if(!res.ok){
        throw new Error(await res.text());
    }
    return res.json();
}

async function postJSON(url,payload){
    const res=await fetch(url,{
        method:"POST",
        headers:{"Content-Type":"application/json"},
        body:JSON.stringify(payload)
    });
    if(!res.ok){
        throw new Error(await res.text());
    }
    return res.json();
}

function formatBytes(value){
    if(!value){
        return "0 B";
    }
    const units=["B","KB","MB","GB","TB","PB"];
    let size=value;
    let index=0;
    while(size>=1024&&index<units.length-1){
        size=size/1024;
        index++;
    }
    return `${size.toFixed(index===0?0:1)} ${units[index]}`;
}

function escapeHTML(value){
    return String(value ?? "").replace(/[&<>"']/g, char => ({
        "&":"&amp;",
        "<":"&lt;",
        ">":"&gt;",
        '"':"&quot;",
        "'":"&#039;"
    }[char]));
}

function renderEdgeOptions(){
    const source=document.querySelector("#source-edge");
    const destination=document.querySelector("#destination-edge");

    const options=cachedEdges.map(edge=>`
        <option value="${escapeHTML(edge.edge_id)}">${escapeHTML(edge.name)} (${escapeHTML(edge.status)})</option>
    `).join("");

    source.innerHTML=options;
    destination.innerHTML=options;
}

async function loadEdges(){

    const edges=(await getJSON("/api/v1/edges")) || [];
    cachedEdges=edges;

    const tbody=document.querySelector("#edges tbody");

    tbody.innerHTML="";

    edges.forEach(edge=>{

        tbody.innerHTML+=`
        <tr>

        <td>${edge.name}</td>

        <td class="${edge.status}">
            ${edge.status}
        </td>

        <td>${edge.ip_address}</td>

        <td>${edge.version}</td>

        <td>${edge.last_seen}</td>

        </tr>
        `;
    });

    renderEdgeOptions();

}

async function loadTransfers(){
    const transfers=(await getJSON("/api/v1/transfers")) || [];
    const tbody=document.querySelector("#transfers tbody");
    tbody.innerHTML="";

    transfers.forEach(job=>{
        const percent=job.bytes_total>0
            ? Math.min(100,(job.bytes_transferred/job.bytes_total)*100)
            : 0;

        tbody.innerHTML+=`
        <tr>
        <td>${escapeHTML(job.job_id)}</td>
        <td class="${escapeHTML(job.status)}">${escapeHTML(job.status)}</td>
        <td>${escapeHTML(job.source_edge_id)}<br><span>${escapeHTML(job.source_path)}</span></td>
        <td>${escapeHTML(job.destination_edge_id)}<br><span>${escapeHTML(job.destination_path)}</span></td>
        <td>
            <progress value="${percent}" max="100"></progress>
            <div>${formatBytes(job.bytes_transferred)} / ${formatBytes(job.bytes_total)}</div>
        </td>
        <td>${escapeHTML(job.worker_id)}</td>
        <td>${escapeHTML(job.error_message)}</td>
        </tr>
        `;
    });
}

async function loadEdgeHealth(){
    const items=(await getJSON("/api/v1/edge-health")) || [];
    const tbody=document.querySelector("#edge-health tbody");
    tbody.innerHTML="";

    items.forEach(item=>{
        const used=item.total_bytes>0
            ? `${formatBytes(item.used_bytes)} / ${formatBytes(item.total_bytes)}`
            : "unknown";

        tbody.innerHTML+=`
        <tr>
        <td>${escapeHTML(item.name || item.edge_id)}<br><span>${escapeHTML(item.status)}</span></td>
        <td>${escapeHTML(item.host)}<br><span>HTTP ${escapeHTML(item.http_port)} / gRPC ${escapeHTML(item.grpc_port)}</span></td>
        <td class="${item.healthy ? "completed" : "failed"}">${item.healthy ? "healthy" : "unhealthy"}<br><span>${escapeHTML(item.mount_path)}</span></td>
        <td>${used}<br><span>${item.writable ? "writable" : "not writable"}</span></td>
        <td>${escapeHTML(item.health_message)}</td>
        <td>${escapeHTML(item.updated_at)}</td>
        </tr>
        `;
    });
}

async function loadWorkers(){
    const workers=(await getJSON("/api/v1/transfer-workers")) || [];
    const tbody=document.querySelector("#workers tbody");
    tbody.innerHTML="";

    workers.forEach(worker=>{
        tbody.innerHTML+=`
        <tr>
        <td>${escapeHTML(worker.name)}</td>
        <td class="${escapeHTML(worker.status)}">${escapeHTML(worker.status)}</td>
        <td>${escapeHTML(worker.current_job_id)}</td>
        <td>${escapeHTML(worker.last_seen)}</td>
        <td><button onclick="deleteWorker('${escapeHTML(worker.id)}')">Delete</button></td>
        </tr>
        `;
    });
}

async function deleteWorker(workerId){
    if(!confirm("Are you sure you want to delete this worker?")){
        return;
    }
    
    try{
        const response=await fetch(`/api/v1/transfer-workers/${workerId}`,{
            method:"DELETE"
        });
        
        if(!response.ok){
            alert("Failed to delete worker");
            return;
        }
        
        await refreshAll();
    }catch(err){
        console.error("Error deleting worker:",err);
        alert("Error deleting worker");
    }
}

async function refreshAll(){
    await Promise.all([loadEdges(),loadEdgeHealth(),loadTransfers(),loadWorkers()]);
}

document.querySelector("#transfer-form").addEventListener("submit",async event=>{
    event.preventDefault();

    await postJSON("/api/v1/transfers",{
        source_edge_id:document.querySelector("#source-edge").value,
        destination_edge_id:document.querySelector("#destination-edge").value,
        source_path:document.querySelector("#source-path").value,
        destination_path:document.querySelector("#destination-path").value,
        chunk_size:Number(document.querySelector("#chunk-size").value)
    });

    event.target.reset();
    await refreshAll();
});

document.querySelector("#worker-form").addEventListener("submit",async event=>{
    event.preventDefault();

    await postJSON("/api/v1/transfer-workers",{
        name:document.querySelector("#worker-name").value
    });

    event.target.reset();
    await refreshAll();
});

document.querySelector("#refresh").addEventListener("click",refreshAll);

refreshAll();

setInterval(refreshAll,5000);
