# typed: strong
# frozen_string_literal: true

module Profile::Followable
  def self.has_many(*args); end
  def follows; end
  def me?(*args); end
end
