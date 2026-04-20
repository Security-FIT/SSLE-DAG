/*
  Warnings:

  - You are about to drop the column `userId` on the `Block` table. All the data in the column will be lost.
  - Added the required column `authorId` to the `Block` table without a default value. This is not possible if the table is not empty.

*/
-- RedefineTables
PRAGMA defer_foreign_keys=ON;
PRAGMA foreign_keys=OFF;
CREATE TABLE "new_Block" (
    "blockHash" TEXT NOT NULL PRIMARY KEY,
    "blockNumber" INTEGER NOT NULL,
    "createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "merkleRoot" TEXT NOT NULL,
    "authorId" TEXT NOT NULL,
    CONSTRAINT "Block_authorId_fkey" FOREIGN KEY ("authorId") REFERENCES "User" ("id") ON DELETE RESTRICT ON UPDATE CASCADE
);
INSERT INTO "new_Block" ("blockHash", "blockNumber", "createdAt", "merkleRoot") SELECT "blockHash", "blockNumber", "createdAt", "merkleRoot" FROM "Block";
DROP TABLE "Block";
ALTER TABLE "new_Block" RENAME TO "Block";
PRAGMA foreign_keys=ON;
PRAGMA defer_foreign_keys=OFF;
