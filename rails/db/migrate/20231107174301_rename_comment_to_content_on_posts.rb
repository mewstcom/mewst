# frozen_string_literal: true

class RenameCommentToContentOnPosts < ActiveRecord::Migration[7.0]
  def change
    StrongMigrations.disable_check(:rename_column)
    rename_column :posts, :comment, :content
  end
end
