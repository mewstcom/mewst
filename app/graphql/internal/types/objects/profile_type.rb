# typed: strict
# frozen_string_literal: true

class Internal::Types::Objects::ProfileType < Internal::Types::Objects::Base
  field :atname, String, null: false
  field :name, String, null: false
end
