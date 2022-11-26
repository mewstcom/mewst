# typed: strong
# frozen_string_literal: true

module SoftDeletable
  def self.scope(*args); end
  def self.where(*args); end
  def deleted_at; end
end
