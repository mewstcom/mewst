# typed: strict
# frozen_string_literal: true

class Internal::ProfileSerializer < Internal::ApplicationSerializer
  attributes :atname, :avatar_url, :id, :name
end
