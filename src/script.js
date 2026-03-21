document.addEventListener('DOMContentLoaded', () => {
    cargarDatos();
});

async function cargarDatos() {
    try {
        const response = await fetch('/api/tarjetas');
        if (!response.ok) throw new Error('Error al obtener los datos de la API');
        
        // Ahora esperamos un objeto con "totales" y "tarjetas"
        const data = await response.json(); 
        renderDashboard(data);

    } catch (error) {
        console.error("Error:", error);
        document.getElementById('tarjetas-container').innerHTML = `
            <p style="color: red; text-align: center; grid-column: 1 / -1;">
                Ocurrió un error al cargar la información: ${error.message}
            </p>`;
    }
}

function renderDashboard(data) {
    const totales = data.totales;
    const tarjetas = data.tarjetas;

    // 1. Actualizar los textos del Resumen General directamente con lo que manda la API
    document.getElementById('total-deuda-header').innerText = formatCurrency(totales.totalUsado);
    document.getElementById('total-ahorro').innerText = formatCurrency(totales.totalAhorro);
    document.getElementById('total-apalancado-grid').innerText = formatCurrency(totales.totalApalancado);
    
    const msiElement = document.getElementById('total-msi-grid');
    if(msiElement) msiElement.innerText = formatCurrency(totales.totalMsi);
    
    document.getElementById('total-disponible').innerText = formatCurrency(totales.totalDisponible);
    document.getElementById('total-utilizacion').innerText = `${totales.utilizacionGlobal.toFixed(1)}%`;

    // 2. Renderizar la Barra General usando los porcentajes que también calculó la API
    const barGeneral = document.getElementById('bar-general');
    if (totales.totalCredito > 0 && barGeneral) {
        barGeneral.innerHTML = `
            <div style="width: ${totales.pctAhorro}%; height: 100%; background-color: var(--color-ahorro);" title="Ahorro: ${formatCurrency(totales.totalAhorro)}"></div>
            <div style="width: ${totales.pctApalancado}%; height: 100%; background-color: var(--color-apalancado);" title="Apalancamiento: ${formatCurrency(totales.totalApalancado)}"></div>
            <div style="width: ${totales.pctMsi}%; height: 100%; background-color: var(--color-msi, #8b5cf6);" title="MSI: ${formatCurrency(totales.totalMsi)}"></div>
            <div style="width: ${totales.pctDisponible}%; height: 100%; background-color: var(--color-disponible);" title="Disponible: ${formatCurrency(totales.totalDisponible)}"></div>
        `;
        barGeneral.style.display = 'flex';
        barGeneral.style.height = '12px';
        barGeneral.style.borderRadius = '6px';
        barGeneral.style.overflow = 'hidden';
    }

    // 3. Renderizar cada Tarjeta (Aquí es donde agregamos lo nuevo del saldo corriente)
    const contenedor = document.getElementById('tarjetas-container');
    contenedor.innerHTML = ''; 

    tarjetas.forEach(t => {
        const card = document.createElement('div');
        card.className = 'tarjeta-card';
        card.style.borderTop = `4px solid ${t.color}`;
        card.style.padding = '15px';
        card.style.backgroundColor = '#fff';
        card.style.borderRadius = '8px';
        card.style.boxShadow = '0 2px 5px rgba(0,0,0,0.05)';

        // Porcentajes individuales para la barrita de la tarjeta (calculados aquí solo para UI, o puedes mandarlos de la API también)
        const pctTener = t.credito > 0 ? (t.tener / t.credito) * 100 : 0;
        const pctApalancamiento = t.credito > 0 ? (t.apalancamiento / t.credito) * 100 : 0;
        const pctMsi = t.credito > 0 ? (t.msi / t.credito) * 100 : 0;
        const pctDisponible = t.credito > 0 ? (t.disponible / t.credito) * 100 : 0;

        card.innerHTML = `
            <div class="tarjeta-header" style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 5px;">
                <h3 style="margin: 0;">${t.nombre}</h3>
                <span style="background-color: ${t.color}20; color: ${t.color}; padding: 4px 8px; border-radius: 12px; font-size: 0.8rem; font-weight: bold;">
                    ${t.usoPorcentaje}% uso
                </span>
            </div>
            
            <div style="font-size: 0.9rem; color: #444; margin-bottom: 12px; font-weight: 600;">
                Crédito: ${formatCurrency(t.credito)}
            </div>
            
            <div class="tarjeta-meta" style="display: flex; justify-content: space-between; font-size: 0.85rem; color: #666; margin-bottom: 15px;">
                <span>Pago: ${t.fechaPago}</span>
                <span>Sem. Actual: ${t.semanaActual}</span>
            </div>

            <div style="display: flex; height: 8px; width: 100%; border-radius: 4px; overflow: hidden; margin-bottom: 15px; background-color: #e2e8f0;">
                <div style="width: ${pctTener}%; background-color: var(--color-ahorro);"></div>
                <div style="width: ${pctApalancamiento}%; background-color: var(--color-apalancado);"></div>
                <div style="width: ${pctMsi}%; background-color: var(--color-msi, #8b5cf6);"></div>
                <div style="width: ${pctDisponible}%; background-color: var(--color-disponible);"></div>
            </div>

            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 10px; font-size: 0.9rem;">
                <div style="display: flex; flex-direction: column;">
                    <span style="color: #666; font-size: 0.75rem;">Ahorro Total</span>
                    <span style="font-weight: bold; color: var(--color-ahorro);">${formatCurrency(t.tener)}</span>
                </div>
                <div style="display: flex; flex-direction: column;">
                    <span style="color: #666; font-size: 0.75rem;">Apalancado</span>
                    <span style="font-weight: bold; color: var(--color-apalancado);">${formatCurrency(t.apalancamiento)}</span>
                </div>
                <div style="display: flex; flex-direction: column;">
                    <span style="color: #666; font-size: 0.75rem;">MSI</span>
                    <span style="font-weight: bold; color: var(--color-msi, #8b5cf6);">${formatCurrency(t.msi)}</span>
                </div>
                <div style="display: flex; flex-direction: column;">
                    <span style="color: #666; font-size: 0.75rem;">Disponible</span>
                    <span style="font-weight: bold; color: var(--color-disponible);">${formatCurrency(t.disponible)}</span>
                </div>
            </div>

            <div style="margin-top: 12px; padding-top: 10px; border-top: 1px dashed #ccc; font-size: 0.8rem; display: flex; justify-content: space-between; color: #555;">
                <div>
                    <span style="display: block; font-size: 0.7rem; color: #888;">Ahorro Saldo Corriente</span>
                    <strong>${formatCurrency(t.ahorroSaldoCorriente)}</strong>
                </div>
                <div style="text-align: right;">
                    <span style="display: block; font-size: 0.7rem; color: #888;">Semana Corriente</span>
                    <strong>Semana ${t.semanaSaldoCorriente}</strong>
                </div>
            </div>
      `;
        contenedor.appendChild(card);
    });
}

function formatCurrency(value) {
    return new Intl.NumberFormat('es-MX', {
        style: 'currency',
        currency: 'MXN'
    }).format(value || 0);
}
