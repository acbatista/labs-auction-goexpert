db = db.getSiblingDB(process.env.MONGO_INITDB_DATABASE || "auction_db");

db.createCollection("users");
db.createCollection("auctions");
db.createCollection("bids");

db.auctions.createIndex({ status: 1, category: 1 });
db.auctions.createIndex({ product_name: "text" });

db.bids.createIndex({ auction_id: 1, amount: -1 });

db.users.insertMany([
    {
        _id: "d290f1ee-6c54-4b01-90e6-d701748f0851",
        name: "Adriano Carvalho Batista",
    },
    {
        _id: "93fb1e9c-523f-4d92-80b4-0f7ba12fef56",
        name: "Wesley Willians",
    },
    {
        _id: "93fb1e9c-523f-4d92-80b4-0f7ba12fef57",
        name: "Sicrano da Silva",
    },
]);