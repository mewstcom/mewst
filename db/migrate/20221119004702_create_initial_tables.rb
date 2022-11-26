# frozen_string_literal: true

class CreateInitialTables < ActiveRecord::Migration[7.0]
  def change
    create_table :accounts, options: "DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci", unsigned: true do |t|
      t.string :email, null: false
      t.integer :sign_in_count, null: false, default: 0
      t.timestamp :current_signed_in_at
      t.timestamp :last_signed_in_at
      t.timestamps
    end
    add_index :accounts, :email, unique: true

    create_table :email_confirmations, options: "DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci", unsigned: true do |t|
      t.references :account, type: :unsigned_bigint
      t.string :email, null: false
      t.integer :event, null: false
      t.string :token, null: false
      t.string :back
      t.datetime :expires_at, null: false
      t.timestamps
    end
    add_index :email_confirmations, :token, unique: true

    create_table :users, options: "DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci", unsigned: true do |t|
      t.references :account, null: false, type: :unsigned_bigint
      t.timestamps
    end

    create_table :organizations, options: "DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci", unsigned: true do |t|
      t.timestamps
    end

    create_table :organization_members, options: "DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci", unsigned: true do |t|
      t.references :organization, null: false, type: :unsigned_bigint
      t.references :user, null: false
      t.timestamps
    end
    add_index :organization_members, %i[organization_id user_id], unique: true

    create_table :profiles, options: "DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci", unsigned: true do |t|
      t.references :profilable, polymorphic: true, null: false, type: :unsigned_bigint
      t.string :idname, null: false, index: {unique: true}
      t.string :name, null: false, default: ""
      t.string :description, null: false, default: ""
      t.timestamp :deleted_at
      t.timestamps
    end
    add_index :profiles, %i[profilable_type profilable_id], unique: true

    create_table :follows, options: "DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci", unsigned: true do |t|
      t.references :source_profile, null: false, type: :unsigned_bigint
      t.references :target_profile, null: false, type: :unsigned_bigint
      t.timestamps
    end
    add_index :follows, %i[source_profile_id target_profile_id], unique: true

    create_table :posts, options: "DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci", unsigned: true do |t|
      t.references :profile, null: false, type: :unsigned_bigint
      t.text :body, null: false
      t.timestamps
    end
  end
end
