document.addEventListener('DOMContentLoaded', () => {
    cargarDashboard();
});

async function cargarDashboard() {
    try {
        const response = await fetch('/api/dashboard/totals');
        
        if (!response.ok) {
            throw new Error('Error al obtener los datos del dashboard');
        }
        
        const totals = await response.json();
        renderDashboardTotales(totals);
    } catch (error) {
        console.error("Error en dashboard:", error);
    }
}

function renderDashboardTotales(totals) {
    document.getElementById('total-deuda-header').innerText = formatCurrency(totals.totalUsado);
    document.getElementById('total-ahorro').innerText = formatCurrency(totals.totalAhorro);
    document.getElementById('total-apalancado-grid').innerText = formatCurrency(totals.totalApalancado);
    
    const msiElement = document.getElementById('total-msi-grid');
    if(msiElement) {
        msiElement.innerText = formatCurrency(totals.totalMsi);
    }

    document.getElementById('total-disponible').innerText = formatCurrency(totals.totalDisponible);
    document.getElementById('total-utilizacion').innerText = `${totals.utilizacionGlobal.toFixed(1)}%`;

    // Renderizado de la barra general
    const barGeneral = document.getElementById('bar-general');
    if (totals.totalCredito > 0 && barGeneral) {
        barGeneral.innerHTML = `
            <div class="tarjeta-barra-segmento bg-ahorro" style="width: ${(totals.totalAhorro / totals.totalCredito) * 100}%; background-color: var(--color-ahorro);" title="Ahorro: ${formatCurrency(totals.totalAhorro)}"></div>
            <div class="tarjeta-barra-segmento bg-apalancado" style="width: ${(totals.totalApalancado / totals.totalCredito) * 100}%;" title="Apalancamiento: ${formatCurrency(totals.totalApalancado)}"></div>
            <div class="tarjeta-barra-segmento bg-msi" style="width: ${(totals.totalMsi / totals.totalCredito) * 100}%;" title="MSI: ${formatCurrency(totals.totalMsi)}"></div>
            <div class="tarjeta-barra-segmento bg-disponible" style="width: ${(totals.totalDisponible / totals.totalCredito) * 100}%;" title="Disponible: ${formatCurrency(totals.totalDisponible)}"></div>
        `;
    }
}

document.addEventListener('DOMContentLoaded', () => {
    const sidebar = document.getElementById('sidebar');
    const btnOpen = document.getElementById('toggle-sidebar-open');
    const btnClose = document.getElementById('toggle-sidebar-close');

    // Función para alternar sidebar
    const toggleSidebar = () => {
        sidebar.classList.toggle('hidden');
    };

    if (btnOpen) btnOpen.onclick = toggleSidebar;
    if (btnClose) btnClose.onclick = toggleSidebar;

    // Si es móvil, ocultar por defecto
    if (window.innerWidth < 1024) {
        sidebar.classList.add('hidden');
    }
    
    cargarDashboard();
    cargarTarjetas();
});