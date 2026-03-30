import { createFileRoute, Navigate } from '@tanstack/react-router'

export const Route = createFileRoute('/')({
	component: () => <Navigate to="/playground" search={{ screen: '', state: '', mode: 'screen', flow: '', step: 0, set: 'wireframe', layout: '', width: 0 }} />,
})
