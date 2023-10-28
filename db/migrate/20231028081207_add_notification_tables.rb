# frozen_string_literal: true

class AddNotificationTables < ActiveRecord::Migration[7.0]
  def change
    create_table :notifications, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :profile, foreign_key: true, null: false, type: :uuid
      t.string :notifiable_type, null: false
      t.timestamp :notified_at, index: true, null: false
      t.timestamp :read_at
      t.timestamps
    end

    create_table :follow_notifications, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :notification, foreign_key: true, null: false, type: :uuid
      t.references :follow, foreign_key: true, index: {unique: true}, null: false, type: :uuid
      t.timestamps
    end

    create_table :stamp_notifications, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :notification, foreign_key: true, null: false, type: :uuid
      t.references :stamp, foreign_key: true, index: {unique: true}, null: false, type: :uuid
      t.timestamps
    end
  end
end
