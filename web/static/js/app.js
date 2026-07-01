async function loadEdges(){

    const res=await fetch("/api/v1/edges");

    const edges=await res.json();

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

}

loadEdges();

setInterval(loadEdges,5000);
