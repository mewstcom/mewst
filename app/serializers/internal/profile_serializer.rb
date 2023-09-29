# typed: strict
# frozen_string_literal: true

class Internal::ProfileSerializer < Internal::ApplicationSerializer
  root_key :profile, :profiles

  attributes :id, :atname, :name, :description, :avatar_url
end
