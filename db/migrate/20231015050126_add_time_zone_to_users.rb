# frozen_string_literal: true

class AddTimeZoneToUsers < ActiveRecord::Migration[7.0]
  def change
    add_column :users, :time_zone, :string, null: false, default: "Etc/UTC"
  end
end
