// chevrotain@12 uses Object.groupBy, which jsdom does not provide.
if (!Object.groupBy) {
    Object.groupBy = (items, keySelector) => {
        const result = Object.create(null);
        for (let i = 0; i < items.length; i++) {
            const key = keySelector(items[i], i);
            if (!Object.prototype.hasOwnProperty.call(result, key)) {
                result[key] = [];
            }
            result[key].push(items[i]);
        }
        return result;
    };
}
