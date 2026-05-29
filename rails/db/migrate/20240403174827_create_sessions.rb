# frozen_string_literal: true

class CreateSessions < ActiveRecord::Migration[7.1]
  def change
    create_table :sessions, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :actor, foreign_key: true, null: false, type: :uuid
      t.string :token, index: {unique: true}, null: false
      t.string :ip_address, default: "", null: false
      t.string :user_agent, default: "", null: false
      t.datetime :signed_in_at, null: false
      t.timestamps
    end
  end
end
