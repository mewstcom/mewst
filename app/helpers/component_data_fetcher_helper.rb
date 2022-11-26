# typed: strict
# frozen_string_literal: true

module ComponentDataFetcherHelper
  extend T::Sig

  sig { params(component_name: String, path: String, payload: T::Hash[Symbol, T.untyped]).returns(String) }
  def component_data_fetcher_tag(component_name, path, payload: {})
    tag.div(
      data: {
        controller: "component-data-fetcher",
        component_data_fetcher_path_value: path,
        component_data_fetcher_event_name_value: "component-data-fetcher:#{component_name}:fetched",
        component_data_fetcher_payload_value: payload
      }
    )
  end
end
