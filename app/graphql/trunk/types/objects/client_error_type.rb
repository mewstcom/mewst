# typed: strict
# frozen_string_literal: true

class Trunk::Types::Objects::ClientErrorType < Trunk::Types::Objects::Base
  field :message, String, null: false
end
