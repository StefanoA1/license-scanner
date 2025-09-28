document.addEventListener('DOMContentLoaded', function() {
    const table = document.getElementById('dependencyTable');
    const headers = table.querySelectorAll('th.sortable');
    let currentSort = { column: -1, direction: 'asc' };

    headers.forEach(header => {
        header.addEventListener('click', function() {
            const column = parseInt(this.dataset.column);
            const direction = currentSort.column === column && currentSort.direction === 'asc' ? 'desc' : 'asc';

            // Remove existing sort classes
            headers.forEach(h => h.classList.remove('asc', 'desc'));

            // Add sort class to clicked header
            this.classList.add(direction);

            sortTable(column, direction);
            currentSort = { column, direction };
        });
    });

    function sortTable(column, direction) {
        const tbody = table.querySelector('tbody');
        const rows = Array.from(tbody.querySelectorAll('tr'));

        rows.sort((a, b) => {
            let aVal = a.cells[column].textContent.trim();
            let bVal = b.cells[column].textContent.trim();

            // Special handling for confidence column (numeric)
            if (column === 3) {
                aVal = parseFloat(aVal);
                bVal = parseFloat(bVal);
                return direction === 'asc' ? aVal - bVal : bVal - aVal;
            }

            // String comparison for other columns
            if (direction === 'asc') {
                return aVal.localeCompare(bVal, undefined, { numeric: true, sensitivity: 'base' });
            } else {
                return bVal.localeCompare(aVal, undefined, { numeric: true, sensitivity: 'base' });
            }
        });

        // Clear tbody and append sorted rows
        tbody.innerHTML = '';
        rows.forEach(row => tbody.appendChild(row));
    }

    // Default sort by package name
    headers[0].click();
});
