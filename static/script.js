let pollInterval;
let parameterRows = [];
let previewChart = null;

document.addEventListener('DOMContentLoaded', function() {
    loadMeshTypes();
    initReactiveTable();
    initPreviewChart();
    document.getElementById('calculateBtn').addEventListener('click', startCalculation);

    // Добавляем обработчики для кнопок
    document.getElementById('addRowBtn').addEventListener('click', addParameterRow);
    document.getElementById('resetTableBtn').addEventListener('click', resetParameterTable);

    // Добавляем обработчики для основных полей ввода
    setupInputListeners();
});

function setupInputListeners() {
    const inputs = ['epsilonStart', 'epsilonMin', 'nStart', 'nMax', 'delta'];
    inputs.forEach(id => {
        document.getElementById(id).addEventListener('input', function() {
            updateTableFromInputs();
        });
    });
}

function initReactiveTable() {
    // Инициализируем с параметрами по умолчанию
    parameterRows = [
        { epsilon: 1.0, n: 128 },
        { epsilon: 0.1, n: 256 },
        { epsilon: 0.01, n: 512 },
        { epsilon: 0.001, n: 1024 },
        { epsilon: 0.0001, n: 2048 }
    ];

    renderParameterTable();
    updateStats();
    updatePreviewChart();
}

function renderParameterTable() {
    const tbody = document.getElementById('paramsTableBody');
    if (!tbody) return;

    tbody.innerHTML = '';

    parameterRows.forEach((row, index) => {
        const tr = document.createElement('tr');

        // Рассчитываем производные значения
        const logEpsilon = Math.log10(row.epsilon).toFixed(2);
        const h = (row.epsilon / row.n).toExponential(2);
        const expectedError = Math.pow(row.n, -0.5).toExponential(2); // Примерная формула

        tr.innerHTML = `
            <td>
                <input type="number" 
                       value="${row.epsilon}" 
                       step="0.1" 
                       min="1e-8"
                       onchange="updateParameter(${index}, 'epsilon', this.value)">
            </td>
            <td>
                <input type="number" 
                       value="${row.n}" 
                       step="1" 
                       min="16"
                       onchange="updateParameter(${index}, 'n', this.value)">
            </td>
            <td>
                <span class="readonly-field">${logEpsilon}</span>
            </td>
            <td>
                <span class="readonly-field">${h}</span>
            </td>
            <td>
                <span class="readonly-field">${expectedError}</span>
            </td>
            <td>
                <button onclick="removeParameterRow(${index})" class="btn-danger" ${parameterRows.length === 1 ? 'disabled' : ''}>❌</button>
            </td>
        `;

        tbody.appendChild(tr);
    });
}

function updateParameter(index, field, value) {
    parameterRows[index][field] = parseFloat(value);

    // Обновляем основные поля ввода
    if (index === 0) {
        document.getElementById('epsilonStart').value = parameterRows[0].epsilon;
        document.getElementById('nStart').value = parameterRows[0].n;
    }
    if (index === parameterRows.length - 1) {
        document.getElementById('epsilonMin').value = parameterRows[parameterRows.length - 1].epsilon;
        document.getElementById('nMax').value = parameterRows[parameterRows.length - 1].n;
    }

    // Перерисовываем таблицу
    renderParameterTable();
    updateStats();
    updatePreviewChart();
}

function updateTableFromInputs() {
    // Синхронизируем таблицу с основными полями ввода
    parameterRows[0] = {
        epsilon: parseFloat(document.getElementById('epsilonStart').value),
        n: parseInt(document.getElementById('nStart').value)
    };

    parameterRows[parameterRows.length - 1] = {
        epsilon: parseFloat(document.getElementById('epsilonMin').value),
        n: parseInt(document.getElementById('nMax').value)
    };

    // Интерполируем промежуточные значения
    const stepEpsilon = Math.pow(
        parameterRows[parameterRows.length - 1].epsilon / parameterRows[0].epsilon,
        1 / (parameterRows.length - 1)
    );

    const stepN = Math.pow(
        parameterRows[parameterRows.length - 1].n / parameterRows[0].n,
        1 / (parameterRows.length - 1)
    );

    for (let i = 1; i < parameterRows.length - 1; i++) {
        parameterRows[i] = {
            epsilon: parameterRows[0].epsilon * Math.pow(stepEpsilon, i),
            n: Math.round(parameterRows[0].n * Math.pow(stepN, i))
        };
    }

    renderParameterTable();
    updateStats();
    updatePreviewChart();
}

function addParameterRow() {
    const lastRow = parameterRows[parameterRows.length - 1];
    parameterRows.push({
        epsilon: lastRow.epsilon / 10,
        n: lastRow.n * 2
    });

    // Обновляем максимальные значения
    document.getElementById('epsilonMin').value = parameterRows[parameterRows.length - 1].epsilon;
    document.getElementById('nMax').value = parameterRows[parameterRows.length - 1].n;

    renderParameterTable();
    updateStats();
    updatePreviewChart();
}

function removeParameterRow(index) {
    if (parameterRows.length > 1) {
        parameterRows.splice(index, 1);

        // Обновляем граничные значения
        document.getElementById('epsilonStart').value = parameterRows[0].epsilon;
        document.getElementById('nStart').value = parameterRows[0].n;
        document.getElementById('epsilonMin').value = parameterRows[parameterRows.length - 1].epsilon;
        document.getElementById('nMax').value = parameterRows[parameterRows.length - 1].n;

        renderParameterTable();
        updateStats();
        updatePreviewChart();
    }
}

function resetParameterTable() {
    initReactiveTable();

    // Сбрасываем основные поля
    document.getElementById('epsilonStart').value = 1.0;
    document.getElementById('epsilonMin').value = 1e-8;
    document.getElementById('nStart').value = 128;
    document.getElementById('nMax').value = 2048;
}

function updateStats() {
    const statsDiv = document.getElementById('previewStats');
    if (!statsDiv) return;

    const delta = parseFloat(document.getElementById('delta').value) || 0;
    const totalPoints = parameterRows.reduce((sum, row) => sum + row.n, 0);
    const avgEpsilon = parameterRows.reduce((sum, row) => sum + row.epsilon, 0) / parameterRows.length;

    statsDiv.innerHTML = `
        <div class="stat-card">
            <div class="stat-label">Всего точек сетки</div>
            <div class="stat-value">${totalPoints.toLocaleString()}</div>
        </div>
        <div class="stat-card">
            <div class="stat-label">Среднее ε</div>
            <div class="stat-value">${avgEpsilon.toExponential(2)}</div>
        </div>
        <div class="stat-card">
            <div class="stat-label">Диапазон N</div>
            <div class="stat-value">${parameterRows[0].n} - ${parameterRows[parameterRows.length-1].n}</div>
        </div>
        <div class="stat-card">
            <div class="stat-label">δ (запаздывание)</div>
            <div class="stat-value">${delta}</div>
        </div>
    `;
}

function initPreviewChart() {
    const ctx = document.getElementById('previewChart')?.getContext('2d');
    if (!ctx) return;

    previewChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: [],
            datasets: [{
                label: 'Теоретическая сходимость',
                data: [],
                borderColor: 'rgb(75, 192, 192)',
                backgroundColor: 'rgba(75, 192, 192, 0.1)',
                tension: 0.4,
                fill: true
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                title: {
                    display: true,
                    text: 'Прогнозируемая сходимость метода'
                },
                tooltip: {
                    callbacks: {
                        label: (context) => {
                            return `Ошибка: ${context.parsed.y.toExponential(2)}`;
                        }
                    }
                }
            },
            scales: {
                x: {
                    type: 'logarithmic',
                    title: {
                        display: true,
                        text: 'N (логарифмическая шкала)'
                    }
                },
                y: {
                    type: 'logarithmic',
                    title: {
                        display: true,
                        text: 'Ошибка (логарифмическая шкала)'
                    }
                }
            }
        }
    });
}

function updatePreviewChart() {
    if (!previewChart) return;

    const delta = parseFloat(document.getElementById('delta').value) || 0;
    const nValues = [];
    const errorValues = [];

    // Берем каждую строку таблицы для графика
    parameterRows.forEach(row => {
        nValues.push(row.n);
        // Теоретическая оценка ошибки O(h^p) где p зависит от δ
        const h = row.epsilon / row.n;
        const estimatedError = Math.pow(h, 1 + Math.abs(delta));
        errorValues.push(estimatedError);
    });

    previewChart.data.labels = nValues;
    previewChart.data.datasets[0].data = errorValues;
    previewChart.update();
}

// Остальные функции (loadMeshTypes, startCalculation, и т.д.) остаются без изменений
async function loadMeshTypes() {
    try {
        const response = await fetch('/api/mesh-types');
        const types = await response.json();

        const select = document.getElementById('meshType');
        types.forEach(type => {
            const option = document.createElement('option');
            option.value = type.id;
            option.textContent = type.name;
            select.appendChild(option);
        });
    } catch (error) {
        console.error('Error loading mesh types:', error);
    }
}

async function startCalculation() {
    // Показать статус
    const statusDiv = document.getElementById('status');
    statusDiv.style.display = 'block';
    statusDiv.className = 'status processing';
    statusDiv.textContent = '⏳ Расчет запущен, ожидайте...';

    // Скрыть предыдущие результаты
    document.getElementById('results').style.display = 'none';

    // Отключить кнопку
    document.getElementById('calculateBtn').disabled = true;

    // Получаем значения и заменяем запятые на точки
    const epsilonStart = document.getElementById('epsilonStart').value.replace(',', '.');
    const epsilonMin = document.getElementById('epsilonMin').value.replace(',', '.');
    const nStart = document.getElementById('nStart').value;
    const nMax = document.getElementById('nMax').value;
    const delta = document.getElementById('delta').value.replace(',', '.');

    const requestData = {
        epsilon_start: parseFloat(epsilonStart),
        epsilon_min: parseFloat(epsilonMin),
        n_start: parseInt(nStart),
        n_max: parseInt(nMax),
        delta: parseFloat(delta),
        mesh_type: document.getElementById('meshType').value
    };

    console.log('Sending request:', requestData);

    try {
        const response = await fetch('/api/calculate', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(requestData)
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const data = await response.json();
        console.log('Response:', data);

        if (data.status === 'processing') {
            startPolling(data.job_id);
        } else {
            showError('Неожиданный ответ от сервера');
        }
    } catch (error) {
        console.error('Error:', error);
        showError('Ошибка при запуске расчета: ' + error.message);
        document.getElementById('calculateBtn').disabled = false;
    }
}

function startPolling(jobId) {
    if (pollInterval) {
        clearInterval(pollInterval);
    }

    pollInterval = setInterval(async () => {
        try {
            const response = await fetch(`/api/status/${jobId}`);
            const data = await response.json();

            if (data.status === 'completed') {
                clearInterval(pollInterval);
                showResults(data);
            } else if (data.status === 'failed') {
                clearInterval(pollInterval);
                showError(data.error || 'Неизвестная ошибка');
            }
        } catch (error) {
            console.error('Polling error:', error);
        }
    }, 2000);
}

function showResults(data) {
    document.getElementById('status').style.display = 'none';
    document.getElementById('calculateBtn').disabled = false;

    const resultsDiv = document.getElementById('results');
    resultsDiv.style.display = 'block';

    document.getElementById('pdfLink').href = data.pdf_url;

    displayTable('classicTable', data.classic, 'Классическая схема');
    displayTable('modifiedTable', data.modified, 'Модифицированная схема');

    createChart(data.classic, data.modified);
}

function displayTable(containerId, data, title) {
    const container = document.getElementById(containerId);

    let html = '<table>';
    html += '<tr><th>ε \\ N</th><th>128</th><th>256</th><th>512</th><th>1024</th><th>2048</th></tr>';

    const epsilons = ['1', '10⁻¹', '10⁻²', '10⁻³', '10⁻⁴', '10⁻⁵', '10⁻⁶', '10⁻⁷', '10⁻⁸'];

    for (let i = 0; i < data.length; i++) {
        html += '<tr>';
        html += `<td>${epsilons[i]}</td>`;
        for (let j = 0; j < data[i].length; j++) {
            html += `<td>${data[i][j]}</td>`;
        }
        html += '</tr>';
    }

    html += '</table>';
    container.innerHTML = html;
}

function createChart(classicData, modifiedData) {
    const ctx = document.getElementById('errorChart').getContext('2d');

    const lastEpsilon = classicData[classicData.length - 1];
    const nValues = [128, 256, 512, 1024, 2048];

    if (window.myChart) {
        window.myChart.destroy();
    }

    window.myChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: nValues,
            datasets: [
                {
                    label: 'Классическая схема',
                    data: lastEpsilon.map(val => parseFloat(val)),
                    borderColor: 'rgb(75, 192, 192)',
                    tension: 0.1
                },
                {
                    label: 'Модифицированная схема',
                    data: modifiedData[modifiedData.length - 1].map(val => parseFloat(val)),
                    borderColor: 'rgb(255, 99, 132)',
                    tension: 0.1
                }
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                x: {
                    type: 'logarithmic',
                    title: {
                        display: true,
                        text: 'N'
                    }
                },
                y: {
                    type: 'logarithmic',
                    title: {
                        display: true,
                        text: 'Ошибка'
                    }
                }
            },
            plugins: {
                title: {
                    display: true,
                    text: 'Сходимость метода (ε = 10⁻⁸)'
                }
            }
        }
    });
}

function showError(message) {
    const statusDiv = document.getElementById('status');
    statusDiv.style.display = 'block';
    statusDiv.className = 'status error';
    statusDiv.textContent = '❌ ' + message;

    document.getElementById('calculateBtn').disabled = false;
}