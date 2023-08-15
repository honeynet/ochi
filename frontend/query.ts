export interface Query {
    id: string; // query id
    content: string; //query content
    owner_id: string; // user id
    active: boolean; // true if a query is applied
    description: string; // description of the query
}
