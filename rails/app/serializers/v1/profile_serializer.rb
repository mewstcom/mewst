# typed: false
# frozen_string_literal: true

class V1::ProfileSerializer < V1::ApplicationSerializer
  root_key :profile, :profiles

  attributes :id, :atname, :name, :description, :avatar_url, :viewer_has_followed
end
