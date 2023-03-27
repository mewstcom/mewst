# typed: true

# DO NOT EDIT MANUALLY
# This is an autogenerated file for dynamic methods in `Google::Api::CustomHttpPattern`.
# Please instead update this file by running `bin/tapioca dsl Google::Api::CustomHttpPattern`.

class Google::Api::CustomHttpPattern
  sig { params(kind: T.nilable(String), path: T.nilable(String)).void }
  def initialize(kind: nil, path: nil); end

  sig { void }
  def clear_kind; end

  sig { void }
  def clear_path; end

  sig { returns(String) }
  def kind; end

  sig { params(value: String).void }
  def kind=(value); end

  sig { returns(String) }
  def path; end

  sig { params(value: String).void }
  def path=(value); end
end