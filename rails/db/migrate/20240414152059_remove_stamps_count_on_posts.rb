# frozen_string_literal: true

class RemoveStampsCountOnPosts < ActiveRecord::Migration[7.1]
  def change
    safety_assured do
      remove_column :posts, :stamps_count, :integer
    end
  end
end
