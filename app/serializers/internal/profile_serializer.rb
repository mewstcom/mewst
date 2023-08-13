# typed: strict
# frozen_string_literal: true

class Internal::ProfileSerializer < Internal::ApplicationSerializer
  root_key :profile, :profiles

  attributes :atname, :avatar_url, :id, :name
end
